BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS "users" (
	"uid"	INTEGER UNIQUE NOT NULL,
	"email"	TEXT UNIQUE NOT NULL,
	"username"	TEXT UNIQUE NOT NULL,
	"role"	TEXT NOT NULL,
	"password"	TEXT NOT NULL,
	"extra"	TEXT NOT NULL,
	PRIMARY KEY("uid" AUTOINCREMENT)
);

CREATE TABLE IF NOT EXISTS "pastes" (
	"uuid"	TEXT UNIQUE NOT NULL,
  "uid"	INTEGER NOT NULL,
	"hash"	INTEGER UNIQUE,
  "password" TEXT NOT NULL,
  "expire_after" DATETIME NOT NULL,
  "access_count" INTEGER NOT NULL,
  "max_access_count" INTEGER NOT NULL,
  "delete_if_not_available" INTEGER NOT NULL,
  "hold_count" INTEGER NOT NULL,
  "hold_before" DATETIME NOT NULL,
	"extra"	TEXT NOT NULL,
  "created_at" DATETIME NOT NULL,
	PRIMARY KEY("uuid")
);

CREATE TABLE IF NOT EXISTS "short_url" (
	"name"	TEXT UNIQUE NOT NULL,
	"target"	TEXT NOT NULL,
	FOREIGN KEY(target) REFERENCES pastes(uuid) ON UPDATE CASCADE ON DELETE CASCADE
	PRIMARY KEY("name")
);

CREATE TABLE IF NOT EXISTS "sessions" (
	"uuid"	TEXT NOT NULL,
  "name" TEXT NOT NULL,
	"value"	TEXT NOT NULL,
	"expire_after"	DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS "webauthn_credentials" (
	"id"	BLOB NOT NULL,
  "user_handle" BLOB NOT NULL,
	"uid"	INTEGER NOT NULL,
	PRIMARY KEY("id"),
	FOREIGN KEY("uid") REFERENCES users(uid) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS  "UID_Mapping" ON "pastes" (
	"uid"	ASC
);

CREATE INDEX IF NOT EXISTS  "UUID" ON "sessions" (
	"uuid"	ASC
);
COMMIT;
