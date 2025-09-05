--------------------------------------------------------------------
-- 1. EXTENSIONS
--------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";   -- full‑text search
CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- extra crypto helpers


--------------------------------------------------------------------
-- 2. CORE TABLES
--------------------------------------------------------------------
CREATE TABLE users (
    id         UUID PRIMARY KEY            DEFAULT uuid_generate_v4(),
    email      TEXT UNIQUE NOT NULL
        CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    name       TEXT
        CHECK (name IS NULL OR length(trim(name)) BETWEEN 1 AND 100),
    avatar_url TEXT
        CHECK (avatar_url IS NULL OR avatar_url ~* '^https?://'),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE otp_tokens (
    id         UUID PRIMARY KEY            DEFAULT uuid_generate_v4(),
    email      TEXT NOT NULL
        CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    token      TEXT NOT NULL CHECK (length(token) = 6 AND token ~ '^[0-9]+$'),
    expires_at TIMESTAMPTZ NOT NULL,
    used       BOOLEAN DEFAULT FALSE,
    attempts   INTEGER DEFAULT 0 CHECK (attempts <= 5),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE boards (
    id               UUID PRIMARY KEY            DEFAULT uuid_generate_v4(),
    title            TEXT NOT NULL
        CHECK (length(trim(title)) BETWEEN 1 AND 200),
    description      TEXT
        CHECK (description IS NULL OR length(description) <= 2000),
    owner_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_board_id  UUID REFERENCES boards(id) ON DELETE CASCADE,
    settings         JSONB DEFAULT '{"theme":"default","auto_archive":false}'
        CHECK (jsonb_typeof(settings) = 'object'),
    version          INTEGER DEFAULT 1 CHECK (version >= 1),
    is_template      BOOLEAN DEFAULT FALSE,
    is_public        BOOLEAN DEFAULT FALSE,
    archived         BOOLEAN DEFAULT FALSE,
    last_activity    TIMESTAMPTZ DEFAULT NOW(),
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT no_self_reference CHECK (id <> parent_board_id)
);

CREATE TABLE board_members (
    id            UUID PRIMARY KEY            DEFAULT uuid_generate_v4(),
    board_id      UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role          TEXT NOT NULL DEFAULT 'member'
        CHECK (role IN ('owner','admin','member','viewer')),
    invited_by    UUID REFERENCES users(id),
    joined_at     TIMESTAMPTZ DEFAULT NOW(),
    last_accessed TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(board_id, user_id)
);

CREATE TABLE columns (
    id        UUID PRIMARY KEY            DEFAULT uuid_generate_v4(),
    board_id  UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    title     TEXT NOT NULL
        CHECK (length(trim(title)) BETWEEN 1 AND 100),
    position  INTEGER NOT NULL DEFAULT 0 CHECK (position >= 0),
    settings  JSONB DEFAULT '{"color":"gray","wip_limit":null}'
        CHECK (jsonb_typeof(settings) = 'object'),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE tasks (
    id               UUID PRIMARY KEY            DEFAULT uuid_generate_v4(),
    title            TEXT NOT NULL
        CHECK (length(trim(title)) BETWEEN 1 AND 300),
    description      TEXT
        CHECK (description IS NULL OR length(description) <= 10000),
    column_id        UUID NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
    board_id         UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    assigned_to      UUID REFERENCES users(id) ON DELETE SET NULL,
    priority         TEXT DEFAULT 'Medium'
        CHECK (priority IN ('Low','Medium','High','Urgent')),
    position         INTEGER NOT NULL DEFAULT 0 CHECK (position >= 0),
    version          INTEGER DEFAULT 1 CHECK (version >= 1),
    deadline         TIMESTAMPTZ,
    completed        BOOLEAN DEFAULT FALSE,
    completed_at     TIMESTAMPTZ,
    tags             TEXT[] DEFAULT ARRAY[]::TEXT[],
    attachments      JSONB DEFAULT '[]' CHECK (jsonb_typeof(attachments) = 'array'),
    nested_board_id  UUID REFERENCES boards(id) ON DELETE SET NULL,
    estimated_hours  DECIMAL(5,2) CHECK (estimated_hours > 0),
    actual_hours     DECIMAL(5,2) CHECK (actual_hours > 0),
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT completed_at_logic CHECK (
        (completed = FALSE AND completed_at IS NULL) OR
        (completed = TRUE  AND completed_at IS NOT NULL)
    )
);

CREATE TABLE realtime_sessions (
    id              UUID PRIMARY KEY            DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    board_id        UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    connection_id   TEXT NOT NULL UNIQUE,
    socket_metadata JSONB DEFAULT '{}' CHECK (jsonb_typeof(socket_metadata) = 'object'),
    last_ping       TIMESTAMPTZ DEFAULT NOW(),
    user_agent      TEXT,
    ip_address      INET,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE user_presence (
    user_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    board_id           UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    cursor_x           INTEGER CHECK (cursor_x >= 0),
    cursor_y           INTEGER CHECK (cursor_y >= 0),
    focused_element    TEXT,
    active_task_id     UUID REFERENCES tasks(id) ON DELETE SET NULL,
    is_typing          BOOLEAN DEFAULT FALSE,
    typing_in_element  TEXT,
    last_activity      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, board_id)
);

CREATE TABLE activity_log (
    id         UUID PRIMARY KEY            DEFAULT uuid_generate_v4(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    board_id   UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    task_id    UUID REFERENCES tasks(id) ON DELETE SET NULL,
    action     TEXT NOT NULL CHECK (action IN (
        'task_create','task_update','task_move','task_delete','task_complete',
        'column_create','column_update','column_delete','column_reorder',
        'board_create','board_update','board_delete',
        'member_invite','member_join','member_remove','member_role_change',
        'comment_create','comment_update','comment_delete'
    )),
    description TEXT NOT NULL,
    metadata   JSONB DEFAULT '{}' CHECK (jsonb_typeof(metadata) = 'object'),
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE comments (
    id        UUID PRIMARY KEY            DEFAULT uuid_generate_v4(),
    task_id   UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content   TEXT NOT NULL
        CHECK (length(trim(content)) BETWEEN 1 AND 2000),
    mentions  UUID[] DEFAULT ARRAY[]::UUID[],
    edited    BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);


--------------------------------------------------------------------
-- 3. INDEXES (no volatile partial indexes that reference NOW())
--------------------------------------------------------------------
CREATE INDEX idx_users_email                ON users(email);
CREATE INDEX idx_boards_owner_created       ON boards(owner_id,created_at DESC) WHERE archived = FALSE;
CREATE INDEX idx_board_members_user_board   ON board_members(user_id,board_id);
CREATE INDEX idx_columns_board_position    ON columns(board_id,position);
CREATE INDEX idx_tasks_column_position     ON tasks(column_id,position) WHERE completed = FALSE;
CREATE INDEX idx_tasks_board_updated       ON tasks(board_id,updated_at DESC);
CREATE INDEX idx_tasks_assigned_pending    ON tasks(assigned_to,deadline) WHERE completed = FALSE;

CREATE INDEX idx_realtime_sessions_board   ON realtime_sessions(board_id,last_ping DESC);
CREATE INDEX idx_user_presence_board       ON user_presence(board_id,last_activity DESC);
CREATE INDEX idx_activity_log_board_recent ON activity_log(board_id,created_at DESC);
CREATE INDEX idx_activity_log_user         ON activity_log(user_id,created_at DESC);
CREATE INDEX idx_comments_task_created     ON comments(task_id,created_at DESC);

-- Full‑text search (immutable functions)
CREATE INDEX idx_tasks_search  ON tasks USING gin (
    to_tsvector('english', coalesce(title,'') || ' ' || coalesce(description,''))
) WHERE completed = FALSE;

CREATE INDEX idx_boards_search ON boards USING gin (
    to_tsvector('english', coalesce(title,'') || ' ' || coalesce(description,''))
) WHERE archived = FALSE;


--------------------------------------------------------------------
-- 4. ENABLE ROW‑LEVEL SECURITY
--------------------------------------------------------------------
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE boards ENABLE ROW LEVEL SECURITY;
ALTER TABLE board_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE columns ENABLE ROW LEVEL SECURITY;
ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE realtime_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_presence ENABLE ROW LEVEL SECURITY;
ALTER TABLE activity_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE otp_tokens ENABLE ROW LEVEL SECURITY;


--------------------------------------------------------------------
-- 5. HELPER FUNCTIONS (used by policies)
--------------------------------------------------------------------
CREATE OR REPLACE FUNCTION user_has_board_access(board_uuid UUID, user_uuid UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1
        FROM boards b
        LEFT JOIN board_members bm ON b.id = bm.board_id
        WHERE b.id = board_uuid
          AND b.archived = FALSE
          AND (b.owner_id = user_uuid OR bm.user_id = user_uuid)
    );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION user_can_modify_board(board_uuid UUID, user_uuid UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1
        FROM boards b
        LEFT JOIN board_members bm ON b.id = bm.board_id
        WHERE b.id = board_uuid
          AND b.archived = FALSE
          AND (
                b.owner_id = user_uuid
                OR (bm.user_id = user_uuid AND bm.role IN ('admin','member'))
          );
    );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;


--------------------------------------------------------------------
-- 6. RLS POLICIES
--------------------------------------------------------------------
-- USERS
CREATE POLICY "All users can read users"
    ON users FOR SELECT TO authenticated USING (TRUE);

CREATE POLICY "Users can update own profile"
    ON users FOR UPDATE TO authenticated USING (auth.uid() = id);

-- OTP TOKENS
CREATE POLICY "Users manage own OTP tokens"
    ON otp_tokens FOR ALL TO authenticated USING (
        email IN (SELECT email FROM users WHERE id = auth.uid())
    );

-- BOARDS
CREATE POLICY "Members can view boards"
    ON boards FOR SELECT TO authenticated USING (
        owner_id = auth.uid()
        OR EXISTS (SELECT 1 FROM board_members WHERE board_id = id AND user_id = auth.uid())
        OR is_public = TRUE
    );

CREATE POLICY "Owners can update boards"
    ON boards FOR UPDATE TO authenticated USING (owner_id = auth.uid());

CREATE POLICY "Owners can delete boards"
    ON boards FOR DELETE TO authenticated USING (owner_id = auth.uid());

CREATE POLICY "Authenticated users can create boards"
    ON boards FOR INSERT TO authenticated WITH CHECK (owner_id = auth.uid());

-- BOARD MEMBERS
CREATE POLICY "Members can view board members"
    ON board_members FOR SELECT TO authenticated USING (
        user_has_board_access(board_id, auth.uid())
    );

CREATE POLICY "Owners & admins can manage members"
    ON board_members FOR ALL TO authenticated USING (
        EXISTS (
            SELECT 1
            FROM boards b
            LEFT JOIN board_members bm ON b.id = bm.board_id
            WHERE b.id = board_id
              AND (b.owner_id = auth.uid()
                   OR (bm.user_id = auth.uid() AND bm.role = 'admin'))
        )
    );

-- COLUMNS
CREATE POLICY "Members can view columns"
    ON columns FOR SELECT TO authenticated USING (
        user_has_board_access(board_id, auth.uid())
    );

CREATE POLICY "Members can modify columns"
    ON columns FOR ALL TO authenticated USING (
        user_can_modify_board(board_id, auth.uid())
    );

-- TASKS
CREATE POLICY "Members can view tasks"
    ON tasks FOR SELECT TO authenticated USING (
        user_has_board_access(board_id, auth.uid())
    );

CREATE POLICY "Members can insert tasks"
    ON tasks FOR INSERT TO authenticated WITH CHECK (
        user_can_modify_board(board_id, auth.uid())
    );

CREATE POLICY "Members can update tasks"
    ON tasks FOR UPDATE TO authenticated USING (
        user_can_modify_board(board_id, auth.uid())
    );

CREATE POLICY "Members can delete tasks"
    ON tasks FOR DELETE TO authenticated USING (
        user_can_modify_board(board_id, auth.uid())
    );

-- REALTIME SESSIONS
CREATE POLICY "Users manage own realtime sessions"
    ON realtime_sessions FOR ALL TO authenticated USING (user_id = auth.uid());

-- USER PRESENCE
CREATE POLICY "Members can view presence"
    ON user_presence FOR SELECT TO authenticated USING (
        user_has_board_access(board_id, auth.uid())
    );

CREATE POLICY "Members can insert own presence"
    ON user_presence FOR INSERT TO authenticated WITH CHECK (user_id = auth.uid());

CREATE POLICY "Members can update own presence"
    ON user_presence FOR UPDATE TO authenticated USING (user_id = auth.uid());

CREATE POLICY "Members can delete own presence"
    ON user_presence FOR DELETE TO authenticated USING (user_id = auth.uid());

-- ACTIVITY LOG
CREATE POLICY "Members can view activity log"
    ON activity_log FOR SELECT TO authenticated USING (
        user_has_board_access(board_id, auth.uid())
    );

CREATE POLICY "System can insert activity log"
    ON activity_log FOR INSERT TO authenticated WITH CHECK (user_id = auth.uid());

-- COMMENTS
CREATE POLICY "Members can view comments"
    ON comments FOR SELECT TO authenticated USING (
        EXISTS (
            SELECT 1
            FROM tasks t
            WHERE t.id = task_id
              AND user_has_board_access(t.board_id, auth.uid())
        )
    );

CREATE POLICY "Members can create comments"
    ON comments FOR INSERT TO authenticated WITH CHECK (
        user_id = auth.uid()
        AND EXISTS (
            SELECT 1
            FROM tasks t
            WHERE t.id = task_id
              AND user_can_modify_board(t.board_id, auth.uid())
        )
    );

CREATE POLICY "Members can update own comments"
    ON comments FOR UPDATE TO authenticated USING (user_id = auth.uid());

CREATE POLICY "Members can delete own comments"
    ON comments FOR DELETE TO authenticated USING (user_id = auth.uid());


--------------------------------------------------------------------
-- 7. TIMESTAMP HELPERS & TRIGGERS
--------------------------------------------------------------------
-- Auto‑update `updated_at` columns
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();