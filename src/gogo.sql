CREATE TABLE moves (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	gameid TEXT NOT NULL, -- the game this move belongs to
	turn INTEGER NOT NULL,   -- the turn number of the move
	player INTEGER NOT NULL, -- 0 = black; 1 = white
	nrow INTEGER NOT NULL,   -- row number of the move (the letter on the board)
	ncol INTEGER NOT NULL,   -- column number of the move (the number on the board)
    ctime INTEGER NOT NULL   -- the time of the move (UNIX timestamp)
);

CREATE TABLE games (
	id TEXT NOT NULL PRIMARY KEY,
	bsize INTEGER NOT NULL,    -- size of the game board
	white TEXT NOT NULL,       -- name of the white player
	black TEXT NOT NULL,       -- name of the black player
	creation INTEGER NOT NULL, -- creation time of the game (UNIX timestamp)
	black_captures INTEGER,    -- number of black stones captured
	white_captures INTEGER,    -- number of white stones captured
	ended INTEGER              -- time the game ended (UNIX timestamp, NULL if not ended yet)
);
