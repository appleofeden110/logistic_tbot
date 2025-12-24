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
    	        references users (chat_id),
    	state      TEXT default 'dormant_mng' not null,
    	check (state in ('dormant_mng', 'waiting_doc', 'waiting_notes', 'waiting_driver'))
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
			car_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			chat_id INTEGER NOT NULL UNIQUE,
			state TEXT DEFAULT 'on_rest' NOT NULL,
			performing_task_id INTEGER,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (chat_id) REFERENCES users(chat_id),
			FOREIGN KEY (car_id) REFERENCES cars(id),
			FOREIGN KEY (performing_task_id) REFERENCES tasks(id),
			CHECK (state IN ('waiting_loc', 'waiting_form_end', 'on_rest', 'performing_load', 'performing_unload', 
			                 'performing_collect', 'performing_dropoff', 'performing_cleaning', 'on_the_road', 
			                 'waiting_photo', 'tracking', 'waiting_km', 'working', 'waiting_attach'))
		)
	`)
	return err
}

func CheckDriversSessionsTable(db DBExecutor) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS drivers_sessions (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			driver_id TEXT NOT NULL,
			date DATETIME DEFAULT CURRENT_DATE,
			started DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
			paused DATETIME,
			worktime TEXT DEFAULT '0h0m' NOT NULL,
			drivetime TEXT DEFAULT '0h0m' NOT NULL,
			pausetime TEXT DEFAULT '0h0m' NOT NULL,
			kilometrage_accumulated INTEGER DEFAULT 0 NOT NULL,
			starting_kilometrage INTEGER,
			end_kilometrage INTEGER,
			FOREIGN KEY (driver_id) REFERENCES drivers(id)
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
			deleted_at TEXT,
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

func CheckFormStatesTable(db DBExecutor) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS form_states (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			chat_id INTEGER,
			user_id TEXT,
			message_text TEXT,
			form_message_id INTEGER,
			which_table TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
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
			doc_id INTEGER,
			started DATETIME,
			finished DATETIME,
			FOREIGN KEY (car_id) REFERENCES cars(id),
			FOREIGN KEY (driver_id) REFERENCES drivers(id),
			FOREIGN KEY (doc_id) REFERENCES files(id),
			CHECK (document_language IN ('fr', 'de', 'en', 'ua', 'pl'))
		)
	`)
	return err
}

func CheckTasksTable(db DBExecutor) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
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
			doc_id INTEGER,
			start TEXT,
			end TEXT,
			current_kilometrage INTEGER,
			current_weight INTEGER,
			current_temperature REAL,
			FOREIGN KEY (shipment_id) REFERENCES shipments(id),
			FOREIGN KEY (doc_id) REFERENCES files(id),
			CHECK (type IN ('load', 'unload', 'collect', 'dropoff', 'cleaning'))
		)
	`)
	return err
}

func CheckTaskDocsTable(db DBExecutor) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS task_docs (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			file_id INTEGER NOT NULL,
			task_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
			FOREIGN KEY (file_id) REFERENCES files(id),
			FOREIGN KEY (task_id) REFERENCES tasks(id)
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
