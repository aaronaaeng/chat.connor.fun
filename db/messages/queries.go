package messages


const (
	createIfNotExistsQuery = `
		CREATE TABLE IF NOT EXISTS messages (
			id UUID NOT NULL PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users (id),
			room_id UUID NOT NULL REFERENCES rooms (id),
			text TEXT NOT NULL,
			create_date Integer NOT NULL
		);
	`

	insertMessageQuery = `
		INSERT INTO messages (id, user_id, room_id, text, create_date) VALUES
			(:id, :user_id, :room_id, :text, :create_date);
	`

	selectOneByIdQuery = `
		SELECT * FROM messages
			WHERE id = :id
	`

	selectByUserIdQuery = `
		SELECT * FROM messages
			WHERE user_id = :user_id;
	`

	selectByRoomIdQuery = `
		SELECT * FROM messages
			WHERE room_id = :user_id;
	`

	selectByUserAndRoomQuery = `
		SELECT * FROM messages
			WHERE user_id = :user_id;
			AND
			room_id = :room_id;
	`

	selectTopByRoomQuery = `
		SELECT * FROM messages
			WHERE room_id = :room_id;
		ORDER BY create_date DESC
		LIMIT :count;
	`

	updateMessageTextQuery = `
		UPDATE messages SET
			text = :text
		WHERE id = :id
		RETURNING *;
	`
)