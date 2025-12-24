package db

func CheckManagersTable(db DBExecutor) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS managers
	(
    	id         TEXT primary key,
    	user_id    TEXT not null unique references users on delete cascade,
    	created_at DATETIME default CURRENT_TIMESTAMP,
    	updated_at DATETIME default CURRENT_TIMESTAMP,
    	chat_id    integer unique
    	    constraint managers___fk
    	        references users (chat_id)
	);
	`)
	return err
}

func CheckCarsTable(db DBExecutor) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS cars (
			id TEXT NOT NULL PRIMARY KEY,
			current_driver TEXT REFERENCES drivers(id),
			current_kilometrage INTEGER NOT NULL
		)
	`)
	return err
}

func CheckCleaningStationsTable(db DBExecutor) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS cleaning_stations (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			address TEXT NOT NULL,
			country TEXT NOT NULL,
			lat REAL NOT NULL,
			lon REAL NOT NULL,
			opening_hours TEXT NOT NULL
		)
	`)
	return err
}

func CheckDriversTable(db DBExecutor) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS drivers (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL UNIQUE,
			car_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			chat_id INTEGER NOT NULL UNIQUE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (chat_id) REFERENCES users(chat_id)
		)
	`)
	return err
}

func CheckFilesTable(db DBExecutor) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			original_name TEXT NOT NULL,
			path TEXT NOT NULL,
			filetype TEXT NOT NULL,
			mimetype TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			deleted_at TEXT DEFAULT CURRENT_TIMESTAMP,
			telegram_file_id TEXT,
			from_chat_id INTEGER,
			FOREIGN KEY (from_chat_id) REFERENCES users(chat_id),
			CHECK (filetype IN ('image', 'video', 'document', 'text', 'other', 'illegal')),
			CHECK (mimetype IN (
				'image/jpeg', 'image/png', 'image/gif', 'image/webp', 'image/bmp', 'image/svg+xml', 'image/tiff',
				'video/mp4', 'video/mpeg', 'video/quicktime', 'video/x-msvideo', 'video/webm', 'video/x-matroska',
				'audio/mpeg', 'audio/mp4', 'audio/aac', 'audio/ogg', 'audio/wav', 'audio/flac',
				'application/zip', 'application/x-rar-compressed', 'application/x-7z-compressed', 'application/x-tar', 'application/gzip',
				'application/pdf', 'application/msword', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
				'application/vnd.ms-excel', 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
				'application/vnd.ms-powerpoint', 'application/vnd.openxmlformats-officedocument.presentationml.presentation',
				'text/plain', 'text/html', 'text/csv', 'text/xml', 'application/json',
				'text/javascript', 'text/css', 'application/javascript',
				'application/octet-stream', 'application/epub+zip', 'application/rtf', 'application/vnd.android.package-archive'
			))
		)
	`)
	return err
}

func CheckShipmentSessionsTable(db DBExecutor) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS shipment_sessions (
			id TEXT NOT NULL PRIMARY KEY,
			driver_id TEXT NOT NULL,
			manager_id TEXT NOT NULL,
			start TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			end TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			shipment_doc_id INTEGER NOT NULL,
			shipment_id INTEGER NOT NULL,
			FOREIGN KEY (driver_id) REFERENCES drivers(id),
			FOREIGN KEY (manager_id) REFERENCES managers(id),
			FOREIGN KEY (shipment_doc_id) REFERENCES files(id),
			FOREIGN KEY (shipment_id) REFERENCES shipments(id)
		)
	`)
	return err
}

func CheckShipmentsTable(db DBExecutor) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS shipments (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			document_language TEXT,
			instruction_type TEXT NOT NULL,
			car_id TEXT NOT NULL,
			driver_id TEXT NOT NULL,
			container TEXT,
			chassis TEXT,
			tankdetails TEXT,
			generalremark TEXT,
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now')),
			FOREIGN KEY (driver_id) REFERENCES drivers(id),
			CHECK (document_language IN ('fr', 'de', 'en', 'ua', 'pl')),
			CHECK (instruction_type IN ('unload instruction', 'load instruction', 'transfer instruction'))
		)
	`)
	return err
}

func CheckTasksTable(db DBExecutor) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id TEXT NOT NULL PRIMARY KEY,
			type TEXT,
			shipment_id INTEGER,
			content TEXT,
			customer_ref TEXT,
			load_ref TEXT,
			load_start_date TEXT DEFAULT (datetime('now')),
			load_end_date TEXT DEFAULT (datetime('now')),
			unload_ref TEXT,
			unload_start_date TEXT DEFAULT (datetime('now')),
			unload_end_date TEXT DEFAULT (datetime('now')),
			tank_status TEXT,
			product TEXT,
			weight TEXT,
			volume TEXT,
			temperature TEXT,
			compartment INTEGER,
			remark TEXT,
			address TEXT,
			destination_address TEXT,
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now')),
			task_doc INTEGER,
			FOREIGN KEY (shipment_id) REFERENCES shipments(id),
			FOREIGN KEY (task_doc) REFERENCES files(id),
			CHECK (type IN ('load', 'unload', 'collect', 'dropoff', 'cleaning'))
		)
	`)
	return err
}

func CheckUsersTable(db DBExecutor) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			chat_id INTEGER NOT NULL UNIQUE,
			name TEXT NOT NULL,
			driver_id TEXT,
			manager_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_super_admin INTEGER DEFAULT 0,
			tg_tag TEXT,
			FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE SET NULL,
			FOREIGN KEY (manager_id) REFERENCES managers(id) ON DELETE SET NULL,
			CHECK (is_super_admin IN (0, 1))
		)
	`)
	return err
}
