package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/kirban/social-media/internal/cache"
	"github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/model"
	"github.com/kirban/social-media/internal/repository"
)

// dialogPageSize is how many of the newest messages a cached dialog page keeps.
// It bounds both the memory one hot dialog can occupy and the work LTRIM does
// on every send.
const dialogPageSize = 500

// dialogIDCacheTTL is long because the pairing a dialog_id resolves to is
// permanent for the lifetime of the two users — nothing ever changes which
// dialog a pair shares, so a stale entry is not a correctness concern, only
// a bound on how long a deleted-dialog edge case could linger.
const dialogIDCacheTTL = 24 * time.Hour

type DialogServiceInterface interface {
	GetMessages(ctx context.Context, srcUser, dstUser model.UserID, limit, offset int64) ([]model.DialogMessage, error)
	SendMessage(ctx context.Context, srcUser, dstUser model.UserID, text string) (*model.DialogMessageID, error)
	ListDialogs(ctx context.Context, srcUser model.UserID) ([]model.DialogSummary, error)
}

type DialogService struct {
	log   *logger.AppLogger
	repo  *repository.DialogRepository
	cache cache.Cache
}

func NewDialogService(l *logger.AppLogger, r *repository.DialogRepository, c cache.Cache) *DialogService {
	return &DialogService{
		log:   l,
		repo:  r,
		cache: c,
	}
}

// GetMessages returns one page of a dialog, newest first.
//
// Only the newest dialogPageSize messages are cached, so a request reaching
// past that window goes straight to Postgres. Within the window the page is
// served from the cached list, and a miss warms it for the next reader. This
// is the path that was saturating at ~6.4k req/s (docs/task7).
func (s *DialogService) GetMessages(ctx context.Context, srcUser, dstUser model.UserID, limit, offset int64) ([]model.DialogMessage, error) {
	dialogID, err := s.resolveDialogID(ctx, srcUser, dstUser)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return []model.DialogMessage{}, nil
		}
		return nil, err
	}

	// A page that straddles the end of the cached window cannot be answered
	// from it, and warming would not help the next reader either.
	if offset+limit > dialogPageSize {
		return s.repo.ListMessages(ctx, dialogID, limit, offset)
	}

	if messages, ok := s.pageFromCache(ctx, *dialogID, limit, offset); ok {
		return messages, nil
	}

	// Miss: fetch the whole cacheable window once, serve the page from it and
	// leave it behind for the next reader. One query, not two.
	window, err := s.repo.ListMessages(ctx, dialogID, dialogPageSize, 0)
	if err != nil {
		return nil, err
	}
	s.warmCachedPage(ctx, *dialogID, window)

	return pageOf(window, limit, offset), nil
}

// resolveDialogID caches the dialog_id lookup for a user pair.
//
// The mapping is permanent once a dialog exists, so unlike the message
// window this is a plain Get/Set — there is no write path that needs to keep
// it warm or append to it, and a hit never needs revalidating against
// Postgres within its TTL.
func (s *DialogService) resolveDialogID(ctx context.Context, srcUser, dstUser model.UserID) (*model.DialogID, error) {
	key := model.DialogIDCacheKey(srcUser, dstUser)

	if cached, err := s.cache.Get(ctx, key); err == nil {
		dialogID := model.DialogID(cached)
		return &dialogID, nil
	} else if !errors.Is(err, cache.ErrCacheMiss) {
		s.log.Error().Err(err).Str("key", key).Msg("dialog: cache read failed")
	}

	dialogID, err := s.repo.GetDialogID(ctx, srcUser, dstUser)
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, key, []byte(*dialogID), dialogIDCacheTTL); err != nil {
		s.log.Error().Err(err).Str("key", key).Msg("dialog: cache write failed")
	}

	return dialogID, nil
}

// pageFromCache answers the request from the cached window, reporting whether
// it could.
func (s *DialogService) pageFromCache(ctx context.Context, dialogID model.DialogID, limit, offset int64) ([]model.DialogMessage, bool) {
	key := model.DialogMessagesCacheKey(dialogID)

	entries, err := s.cache.GetList(ctx, key)
	if err != nil {
		if !errors.Is(err, cache.ErrCacheMiss) {
			s.log.Error().Err(err).Str("key", key).Msg("dialog: cache read failed")
		}
		return nil, false
	}

	messages, ok := decodePage(entries, limit, offset)
	if !ok {
		return nil, false
	}
	if messages == nil {
		// Corrupt entry: drop the page rather than serve a short one.
		s.log.Error().Str("key", key).Msg("dialog: cached page undecodable, dropping")
		if err := s.cache.Delete(ctx, key); err != nil {
			s.log.Error().Err(err).Str("key", key).Msg("dialog: dropping bad page failed")
		}
		return nil, false
	}

	return messages, true
}

// decodePage slices a newest-first page out of a cached window.
//
// entries are stored oldest-first, matching the order dialog_append pushes new
// messages onto the tail, so the newest message is the last element and the
// requested page has to be walked backwards.
//
// The second return reports whether the window covers the page at all; a nil
// first return with ok=true means an entry failed to decode.
func decodePage(entries [][]byte, limit, offset int64) ([]model.DialogMessage, bool) {
	n := int64(len(entries))
	if limit <= 0 || offset < 0 || offset+limit > n {
		return nil, false
	}

	messages := make([]model.DialogMessage, 0, limit)
	for i := n - 1 - offset; i > n-1-offset-limit; i-- {
		var m model.DialogMessage
		if err := json.Unmarshal(entries[i], &m); err != nil {
			return nil, true
		}
		messages = append(messages, m)
	}

	return messages, true
}

// warmCachedPage stores the newest window so later reads, and the append on
// every send, have something to work with.
//
// newestFirst arrives in query order and is reversed on the way in: the list
// is kept oldest-first so that dialog_append can RPUSH onto the tail and LTRIM
// the oldest away.
func (s *DialogService) warmCachedPage(ctx context.Context, dialogID model.DialogID, newestFirst []model.DialogMessage) {
	if len(newestFirst) == 0 {
		return
	}

	key := model.DialogMessagesCacheKey(dialogID)

	values := make([][]byte, len(newestFirst))
	for i, m := range newestFirst {
		payload, err := json.Marshal(m)
		if err != nil {
			s.log.Error().Err(err).Str("key", key).Msg("dialog: marshal message for cache warm")
			return
		}
		values[len(newestFirst)-1-i] = payload
	}

	if err := s.cache.SetList(ctx, key, values, cache.DefaultTTL); err != nil {
		s.log.Error().Err(err).Str("key", key).Msg("dialog: cache warm failed")
	}
}

// pageOf slices a page out of an already newest-first result set.
func pageOf(messages []model.DialogMessage, limit, offset int64) []model.DialogMessage {
	n := int64(len(messages))
	if offset >= n {
		return []model.DialogMessage{}
	}
	end := offset + limit
	if end > n {
		end = n
	}
	return messages[offset:end]
}

// ListDialogs returns the conversations srcUser takes part in.
func (s *DialogService) ListDialogs(ctx context.Context, srcUser model.UserID) ([]model.DialogSummary, error) {
	return s.repo.ListDialogs(ctx, srcUser)
}

func (s *DialogService) SendMessage(ctx context.Context, srcUser, dstUser model.UserID, text string) (*model.DialogMessageID, error) {
	var dialogID *model.DialogID
	dialogID, err := s.resolveDialogID(ctx, srcUser, dstUser)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			ids := []string{srcUser, dstUser}
			slices.Sort(ids)
			var users [2]model.UserID
			copy(users[:], ids)
			dto := &model.CreateDialogDTO{
				Users: users,
			}
			newDID, err := s.repo.CreateDialog(ctx, dto)
			if err != nil {
				return nil, err
			}
			dialogID = newDID

			key := model.DialogIDCacheKey(srcUser, dstUser)
			if err := s.cache.Set(ctx, key, []byte(*dialogID), dialogIDCacheTTL); err != nil {
				s.log.Error().Err(err).Str("key", key).Msg("dialog: cache write failed")
			}
		} else {
			return nil, err
		}
	}

	msg, err := s.repo.CreateMessage(ctx, dialogID, srcUser, dstUser, text)
	if err != nil {
		return nil, err
	}

	s.appendToCachedPage(ctx, *dialogID, *msg)

	return &msg.ID, nil
}

// appendToCachedPage pushes a just-written message onto the dialog's cached
// page instead of invalidating it.
//
// Invalidating here would be self-defeating: a busy dialog is written to
// constantly, so the page would be dropped before any reader could use it and
// the cache would never be warm. Appending keeps a hot dialog hot.
//
// A cold page stays cold — the Lua function's EXISTS guard refuses to create
// one — because a page seeded from a single message would look complete to
// readers while missing all prior history. Warming is the read path's job.
//
// Failures are logged, never returned: the message is already committed, so
// the send succeeded. A lost append only costs a cache miss, and the page's
// TTL bounds how long the cache can stay stale.
func (s *DialogService) appendToCachedPage(ctx context.Context, dialogID model.DialogID, msg model.DialogMessage) {
	key := model.DialogMessagesCacheKey(dialogID)

	payload, err := json.Marshal(msg)
	if err != nil {
		s.log.Error().Err(err).Str("key", key).Msg("dialog: marshal message for cache append")
		return
	}

	appended, err := s.cache.AppendIfWarm(ctx, key, payload, dialogPageSize, cache.DefaultTTL)
	if err != nil {
		s.log.Error().Err(err).Str("key", key).Msg("dialog: cache append failed")
		return
	}
	if !appended {
		s.log.Debug().Str("key", key).Msg("dialog: cache page cold, append skipped")
	}
}
