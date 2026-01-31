package docs

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	ErrNoPermission error = errors.New("you do not have permission to access these files")
)

type File struct {
	Id           int       `db:"id"`
	TgFileId     string    `db:"telegram_file_id"`
	From         int64     `db:"from_chat_id"`
	Name         string    `db:"name"`
	OriginalName string    `db:"original_name"`
	Path         string    `db:"path"` // should be either from server, relative from the bot's executable location or a url (S3)
	Mimetype     Mimetype  `db:"mimetype"`
	Filetype     Filetype  `db:"filetype"`
	CreatedAt    time.Time `db:"created_at"`
	DeletedAt    time.Time `db:"deleted_at"`
}

func (f *File) StoreFile(globalStorage *sql.DB) error {
	query := `
	INSERT INTO files 
		(
		telegram_file_id,
		from_chat_id, 
		name, 
		original_name, 	
		path, 
		filetype, 
		mimetype, 
		created_at
	) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := globalStorage.Exec(
		query,
		f.TgFileId,
		f.From,
		f.Name,
		f.OriginalName,
		f.Path,
		string(f.Filetype),
		string(f.Mimetype),
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("insert file: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	f.Id = int(id)
	f.CreatedAt = time.Now()

	return nil
}

func GetFilesAttachedToTask(globalStorage *sql.DB, taskId int) ([]*File, error) {
	query := `
		SELECT
			f.id,
			f.telegram_file_id,
			f.from_chat_id,
			f.name,
			f.original_name,
			f.path,
			f.mimetype,
			f.filetype,
			f.created_at,
			f.deleted_at
		FROM files f
		INNER JOIN task_docs td ON td.file_id = f.id
		WHERE td.task_id = ?
		  AND f.deleted_at IS NULL
	`

	rows, err := globalStorage.Query(query, taskId)
	if err != nil {
		return nil, fmt.Errorf("query files attached to task %d: %w", taskId, err)
	}
	defer rows.Close()

	files := make([]*File, 0)

	for rows.Next() {
		f := &File{}

		var (
			createdAtStr string
			deletedAtStr sql.NullString
			mimetypeStr  string
			filetypeStr  string
		)

		err := rows.Scan(
			&f.Id,
			&f.TgFileId,
			&f.From,
			&f.Name,
			&f.OriginalName,
			&f.Path,
			&mimetypeStr,
			&filetypeStr,
			&createdAtStr,
			&deletedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}

		// Parse created_at
		if createdAtStr != "" {
			t, err := time.Parse("2006-01-02 15:04:05.999999-07:00", createdAtStr)
			if err != nil {
				return nil, fmt.Errorf("parse created_at: %w", err)
			}
			f.CreatedAt = t
		}

		// Parse deleted_at (nullable)
		if deletedAtStr.Valid {
			t, err := time.Parse("2006-01-02 15:04:05.999999-07:00", deletedAtStr.String)
			if err != nil {
				return nil, fmt.Errorf("parse deleted_at: %w", err)
			}
			f.DeletedAt = t
		}

		f.Filetype = Filetype(filetypeStr)
		f.Mimetype = Mimetype(mimetypeStr)

		files = append(files, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return files, nil
}

func (f *File) AttachFileToTask(globalStorage *sql.DB, taskId int) error {
	query := `INSERT INTO task_docs
	(file_id, task_id) 
	VALUES (?, ?)`

	result, err := globalStorage.Exec(
		query,
		f.Id, taskId,
	)
	if err != nil {
		return fmt.Errorf("insert file: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	f.Id = int(id)
	f.CreatedAt = time.Now()

	return nil
}

// Will fill in any missing details from the db. Gotta have either TgFileId, Name or Id
func (f *File) GetFile(globalStorage *sql.DB) error {
	var query string
	var arg interface{}

	switch {
	case f.Id != 0:
		query = "SELECT id, telegram_file_id, from_chat_id, name, original_name, path, filetype, mimetype, created_at, deleted_at FROM files WHERE id = ?"
		arg = f.Id
	case f.TgFileId != "":
		query = "SELECT id, telegram_file_id, from_chat_id, name, original_name, path, filetype, mimetype, created_at, deleted_at FROM files WHERE telegram_file_id = ?"
		arg = f.TgFileId
	case f.Name != "":
		query = "SELECT id, telegram_file_id, from_chat_id, name, original_name, path, filetype, mimetype, created_at, deleted_at FROM files WHERE name = ?"
		arg = f.Name
	default:
		return fmt.Errorf("must provide either Id, TgFileId, or Name")
	}

	row := globalStorage.QueryRow(query, arg)

	var filetypeStr string
	var mimetypeStr string
	var createdAtStr string
	var deletedAtStr sql.NullString

	if err := row.Scan(
		&f.Id,
		&f.TgFileId,
		&f.From,
		&f.Name,
		&f.OriginalName,
		&f.Path,
		&filetypeStr,
		&mimetypeStr,
		&createdAtStr,
		&deletedAtStr,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("file not found")
		}
		return fmt.Errorf("scan file: %w", err)
	}

	if createdAtStr != "" {
		t, err := time.Parse("2006-01-02 15:04:05.999999-07:00", createdAtStr)
		if err != nil {
			return fmt.Errorf("parse created_at: %w", err)
		}
		f.CreatedAt = t
	}
	if deletedAtStr.Valid {
		t, err := time.Parse("2006-01-02 15:04:05.999999-07:00", deletedAtStr.String)
		if err != nil {
			return fmt.Errorf("parse deleted_at: %w", err)
		}
		f.DeletedAt = t
	}

	f.Filetype = Filetype(filetypeStr)
	f.Mimetype = Mimetype(mimetypeStr)

	return nil
}

// must have query with "getallfiles:<chat_id>, and to check if the person can actually do this, we are checking if he is super admin or not, otherwise, it is impossible to get someone else's files"
func GetAllFilesFromUser(globalStorage *sql.DB, query tgbotapi.CallbackQuery) ([]*File, error) {
	var (
		isSuperAdmin bool
		chatId       int64
		err          error
	)

	chatIdString, f := strings.CutPrefix(query.Data, "getallfiles:")
	if !f {
		return nil, fmt.Errorf("wrong query")
	}

	chatId, err = strconv.ParseInt(chatIdString, 10, 64)
	if err != nil {
		return nil, err
	}

	var superAdminTemp int
	userRow := globalStorage.QueryRow("select users.is_super_admin from users where chat_id = $1", query.Message.Chat.ID)
	if err := userRow.Scan(&superAdminTemp); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("how are you a user, I could not find you")
		}
		return nil, fmt.Errorf("scan admin: %w", err)
	}

	isSuperAdmin = superAdminTemp != 0

	if query.Message.Chat.ID != chatId && !isSuperAdmin {
		return nil, ErrNoPermission
	}

	stmt, err := globalStorage.Prepare("select * from files where from_chat_id = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(chatId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		f := &File{}
		var filetypeStr string
		var mimetypeStr string

		err := rows.Scan(
			&f.Id,
			&f.TgFileId,
			&f.From,
			&f.Name,
			&f.OriginalName,
			&f.Path,
			&filetypeStr,
			&mimetypeStr,
			&f.CreatedAt,
			&f.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		// Convert string to custom types
		f.Filetype = Filetype(filetypeStr)
		f.Mimetype = Mimetype(mimetypeStr)

		files = append(files, f)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return files, nil
}

// Path or FileID of the file should be present
func (f *File) SendFileTo(caption string, markup tgbotapi.InlineKeyboardMarkup, from, to int64, bot tgbotapi.BotAPI) (tgbotapi.Message, error) {
	if f.From == 0 {
		f.From = from
	}

	doc := tgbotapi.NewDocument(to, tgbotapi.FileID(f.TgFileId))
	doc.Caption = caption
	doc.ReplyMarkup = markup
	doc.ParseMode = tgbotapi.ModeHTML

	return bot.Send(doc)
}
