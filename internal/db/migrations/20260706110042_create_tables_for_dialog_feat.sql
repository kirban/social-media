-- +goose Up
CREATE TABLE IF NOT EXISTS "dialog" (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "dialog_user" (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	dialog_id UUID NOT NULL,
	user_id UUID,

	CONSTRAINT fk_dialoguser_dialog
		FOREIGN KEY (dialog_id)
		REFERENCES "dialog"(id)
		ON DELETE CASCADE,

	CONSTRAINT fk_dialoguser_user
		FOREIGN KEY (user_id)
		REFERENCES "users"(id)
		ON DELETE SET NULL
);

CREATE INDEX idx_dialog_user_user_id ON "dialog_user"(user_id);
CREATE UNIQUE INDEX idx_unique_dialog_user ON "dialog_user"(dialog_id, user_id) WHERE user_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS "dialog_message" (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	dialog_id UUID NOT NULL,
	"from" UUID,
	"to" UUID,
	"text" TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

	CONSTRAINT fk_message_dialog
		FOREIGN KEY (dialog_id)
		REFERENCES "dialog"(id)
		ON DELETE CASCADE,

	CONSTRAINT fk_message_sender
		FOREIGN KEY ("from") 
		REFERENCES "users"(id) 
		ON DELETE SET NULL,
	
	CONSTRAINT fk_message_receiver
		FOREIGN KEY ("to") 
		REFERENCES "users"(id) 
		ON DELETE SET NULL
);

CREATE INDEX idx_dialog_message ON "dialog_message" (dialog_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS "dialog";
DROP TABLE IF EXISTS "dialog_user";
DROP TABLE IF EXISTS "dialog_message";
DROP INDEX IF EXISTS "idx_dialog_user_user_id";
DROP INDEX IF EXISTS "idx_unique_dialog_user";
DROP INDEX IF EXISTS "idx_dialog_message";