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
CREATE INDEX "UID_Mapping" ON "pastes" (
	"uid"	ASC
);
COMMIT;
