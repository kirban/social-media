#!lua name=my_utils

-- There is deliberately no delete-by-prefix function here.
--
-- In cluster mode Redis rejects any script whose accessed keys span more than
-- one slot — "Script attempted to access keys that do not hash to the same
-- slot" — and the check is per script *execution*, not per command, so
-- unlinking one key at a time does not help either. A node owns ~5461 slots
-- and SCAN returns keys from all of them, so a SCAN-then-UNLINK function can
-- never be valid. That loop lives in RedisCache.DeleteByPrefix instead, where
-- each UNLINK is its own single-key command.

-- dialog_append appends one message to a cached dialog page, trims the page to
-- its newest maxlen entries and refreshes its TTL.
--
-- keys[1] — the list key
-- args[1] — the encoded message
-- args[2] — maxlen, the number of newest entries to keep
-- args[3] — ttl in seconds
--
-- Returns 1 if the message was appended, 0 if the key was cold.
--
-- The EXISTS guard is the reason this is a function and not a MULTI/EXEC
-- block: a transaction cannot branch on a value it reads. Without the guard a
-- send against a cold key would create a one-element list that looks like a
-- complete page, and readers would silently lose the rest of the history.
--
-- RPUSH + LTRIM + EXPIRE must also be atomic — interleaved senders would
-- otherwise leave the list transiently over-length or without a TTL.
redis.register_function('dialog_append', function(keys, args)
    local key = keys[1]

    if redis.call('EXISTS', key) == 0 then
        return 0
    end

    redis.call('RPUSH', key, args[1])
    redis.call('LTRIM', key, -tonumber(args[2]), -1)
    redis.call('EXPIRE', key, tonumber(args[3]))

    return 1
end)
