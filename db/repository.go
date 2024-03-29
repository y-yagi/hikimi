package db

import (
	"github.com/jmoiron/sqlx"

	"github.com/y-yagi/goext/osext"
)

var (
	schema = `
CREATE TABLE musics (
	id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	key varchar NOT NULL,
	title varchar,
	album varchar,
	artist varchar
	bucket varchar
);

CREATE UNIQUE INDEX KEYINDEX ON musics(key);
`
)

type Music struct {
	ID     int    `db:"id"`
	Key    string `db:"key"`
	Title  string `db:"title"`
	Album  string `db:"album"`
	Artist string `db:"artist"`
	Bucket string `db:"bucket"`
}

type Repository struct {
	database string
}

func NewRepository(database string) *Repository {
	return &Repository{database: database}
}

func (r *Repository) InitDB() error {
	if osext.IsExist(r.database) {
		return nil
	}

	db, err := sqlx.Connect("sqlite3", r.database)
	if err != nil {
		return err
	}
	defer db.Close()

	db.MustExec(schema)

	return nil
}

func (r *Repository) Insert(musics []*Music) error {
	db, err := sqlx.Connect("sqlite3", r.database)
	if err != nil {
		return err
	}
	defer db.Close()

	tx := db.MustBegin()
	for _, music := range musics {
		_, err = tx.NamedExec("INSERT INTO musics (key, title, album, artist, bucket) VALUES (:key, :title, :album, :artistm :bucket)", music)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()

	return nil
}

func (r *Repository) Exist(key string) bool {
	db, _ := sqlx.Connect("sqlite3", r.database)
	defer db.Close()

	rows, _ := db.NamedQuery(`SELECT * FROM musics WHERE key=:key`, map[string]interface{}{"key": key})
	return rows.Next()
}

func (r *Repository) Search(text, bucket string) ([]Music, error) {
	db, _ := sqlx.Connect("sqlite3", r.database)
	defer db.Close()

	rows, _ := db.NamedQuery(`SELECT * FROM musics WHERE bucket = :bucket AND key LIKE :text`, map[string]interface{}{"bucket": bucket, "text": "%" + text + "%"})

	musics := []Music{}
	for rows.Next() {
		var music Music
		err := rows.StructScan(&music)
		if err != nil {
			return musics, err
		}
		musics = append(musics, music)
	}

	return musics, nil
}
