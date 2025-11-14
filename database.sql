--------------------------------------------------------------------
-- 1. EXTENSIONS
--------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE SCHEMA IF NOT EXISTS extensions;
CREATE EXTENSION IF NOT EXISTS "pg_trgm" WITH SCHEMA extensions;   -- full‑text search
CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- extra crypto helpers


--------------------------------------------------------------------
-- 2. CORE TABLES
--
-- SECURITY NOTE: This schema supports encrypted data storage
-- - User emails are encrypted with AES-256-GCM (deterministic for queries)
-- - OTP tokens are hashed with Argon2id + random salt (one-way)
-- - All sensitive data is encrypted at rest in the database
--------------------------------------------------------------------
CREATE TABLE users (
    id         UUID PRIMARY KEY            DEFAULT uuid_generate_v4(),
    email      TEXT UNIQUE NOT NULL
        CHECK (length(email) > 0 AND length(email) <= 500),  -- Encrypted email storage
    name       TEXT
        CHECK (name IS NULL OR length(trim(name)) BETWEEN 1 AND 100),
    avatar_url TEXT
        CHECK (avatar_url IS NULL OR avatar_url ~* '^(https?://|data:image/)'),
    onboarding_completed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE otp_tokens (
    id         UUID PRIMARY KEY            DEFAULT uuid_generate_v4(),
    email      TEXT NOT NULL
        CHECK (length(email) > 0 AND length(email) <= 500),  -- Encrypted email storage
    token      TEXT NOT NULL
        CHECK (length(token) > 0 AND length(token) <= 500),  -- Hashed OTP storage
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
    role          TEXT NOT NULL DEFAULT 'collaborator'
        CHECK (role IN ('owner','admin','collaborator','viewer')),
    invited_by    UUID REFERENCES users(id) ON DELETE SET NULL,
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

--------------------------------------------------------------------
-- 3. APPROVAL WORKFLOW TABLES
--------------------------------------------------------------------

-- Stores proposed changes that require approval from owner/admin
CREATE TABLE proposed_edits (
    id             UUID PRIMARY KEY            DEFAULT uuid_generate_v4(),
    resource_type  TEXT NOT NULL CHECK (resource_type IN ('task','column','board')),
    resource_id    UUID NOT NULL,
    operation_type TEXT NOT NULL CHECK (operation_type IN ('create','update','delete','move')),
    proposed_by    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    board_id       UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    payload        JSONB NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    original_data  JSONB CHECK (jsonb_typeof(original_data) = 'object'),
    status         TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','approved','rejected','applied')),
    reviewer_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    review_reason  TEXT,
    reviewed_at    TIMESTAMPTZ,
    expires_at     TIMESTAMPTZ DEFAULT (NOW() + INTERVAL '7 days'),
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW()
);

-- Notification system for pending approvals
CREATE TABLE approval_notifications (
    id               UUID PRIMARY KEY            DEFAULT uuid_generate_v4(),
    proposed_edit_id UUID NOT NULL REFERENCES proposed_edits(id) ON DELETE CASCADE,
    recipient_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at          TIMESTAMPTZ,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(proposed_edit_id, recipient_id)
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
        'comment_create','comment_update','comment_delete',
        'edit_proposed','edit_approved','edit_rejected','edit_applied'
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
-- 4. INDEXES
--------------------------------------------------------------------
CREATE INDEX idx_users_email                ON users(email);
CREATE INDEX idx_boards_owner_created       ON boards(owner_id,created_at DESC) WHERE archived = FALSE;
CREATE INDEX idx_board_members_user_board   ON board_members(user_id,board_id);
CREATE INDEX idx_board_members_role         ON board_members(board_id,role);
CREATE INDEX idx_columns_board_position     ON columns(board_id,position);
CREATE INDEX idx_tasks_column_position      ON tasks(column_id,position) WHERE completed = FALSE;
CREATE INDEX idx_tasks_board_updated        ON tasks(board_id,updated_at DESC);
CREATE INDEX idx_tasks_assigned_pending     ON tasks(assigned_to,deadline) WHERE completed = FALSE;

-- Approval workflow indexes
CREATE INDEX idx_proposed_edits_status       ON proposed_edits(status,created_at DESC);
CREATE INDEX idx_proposed_edits_board        ON proposed_edits(board_id,status);
CREATE INDEX idx_proposed_edits_proposed_by  ON proposed_edits(proposed_by,status);
CREATE INDEX idx_proposed_edits_resource     ON proposed_edits(resource_type,resource_id,status);
CREATE INDEX idx_proposed_edits_expires      ON proposed_edits(expires_at) WHERE status = 'pending';

CREATE INDEX idx_approval_notifications_recipient ON approval_notifications(recipient_id,read_at);

CREATE INDEX idx_realtime_sessions_board     ON realtime_sessions(board_id,last_ping DESC);
CREATE INDEX idx_user_presence_board         ON user_presence(board_id,last_activity DESC);
CREATE INDEX idx_activity_log_board_recent   ON activity_log(board_id,created_at DESC);
CREATE INDEX idx_activity_log_user           ON activity_log(user_id,created_at DESC);
CREATE INDEX idx_comments_task_created       ON comments(task_id,created_at DESC);

-- Additional foreign key indexes for performance
CREATE INDEX idx_activity_log_task_id        ON activity_log(task_id);
CREATE INDEX idx_board_members_invited_by    ON board_members(invited_by);
CREATE INDEX idx_boards_parent_board_id      ON boards(parent_board_id);
CREATE INDEX idx_comments_user_id            ON comments(user_id);
CREATE INDEX idx_proposed_edits_reviewer_id  ON proposed_edits(reviewer_id);
CREATE INDEX idx_realtime_sessions_user_id   ON realtime_sessions(user_id);
CREATE INDEX idx_tasks_nested_board_id       ON tasks(nested_board_id);
CREATE INDEX idx_user_presence_active_task_id ON user_presence(active_task_id);

-- Full‑text search
CREATE INDEX idx_tasks_search  ON tasks USING gin (
    to_tsvector('english', coalesce(title,'') || ' ' || coalesce(description,''))
) WHERE completed = FALSE;

CREATE INDEX idx_boards_search ON boards USING gin (
    to_tsvector('english', coalesce(title,'') || ' ' || coalesce(description,''))
) WHERE archived = FALSE;


--------------------------------------------------------------------
-- 5. ENABLE ROW‑LEVEL SECURITY
--------------------------------------------------------------------
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE boards ENABLE ROW LEVEL SECURITY;
ALTER TABLE board_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE columns ENABLE ROW LEVEL SECURITY;
ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE proposed_edits ENABLE ROW LEVEL SECURITY;
ALTER TABLE approval_notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE realtime_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_presence ENABLE ROW LEVEL SECURITY;
ALTER TABLE activity_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE otp_tokens ENABLE ROW LEVEL SECURITY;


--------------------------------------------------------------------
-- 6. HELPER FUNCTIONS
--------------------------------------------------------------------

-- Check if user has board access
CREATE OR REPLACE FUNCTION public.user_has_board_access(board_uuid UUID, user_uuid UUID)
RETURNS BOOLEAN
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1
        FROM public.boards b
        LEFT JOIN public.board_members bm ON b.id = bm.board_id
        WHERE b.id = board_uuid
          AND b.archived = FALSE
          AND (b.owner_id = user_uuid OR bm.user_id = user_uuid)
    );
END;
$$;

-- Check if user can modify board without approval (owner/admin)
CREATE OR REPLACE FUNCTION public.user_can_modify_directly(board_uuid UUID, user_uuid UUID)
RETURNS BOOLEAN
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1
        FROM public.boards b
        LEFT JOIN public.board_members bm ON b.id = bm.board_id
        WHERE b.id = board_uuid
          AND b.archived = FALSE
          AND (
                b.owner_id = user_uuid
                OR (bm.user_id = user_uuid AND bm.role = 'admin')
          )
    );
END;
$$;

-- Check if user can approve edits (owner/admin)
CREATE OR REPLACE FUNCTION public.user_can_approve_edits(board_uuid UUID, user_uuid UUID)
RETURNS BOOLEAN
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
BEGIN
    RETURN public.user_can_modify_directly(board_uuid, user_uuid);
END;
$$;

-- Check if user can manage members (owner for admins, owner/admin for collaborators/viewers)
CREATE OR REPLACE FUNCTION public.user_can_manage_member(board_uuid UUID, manager_uuid UUID, target_role TEXT)
RETURNS BOOLEAN
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
DECLARE
    manager_role TEXT;
BEGIN
    -- Get manager's role
    SELECT CASE
        WHEN b.owner_id = manager_uuid THEN 'owner'
        ELSE COALESCE(bm.role, 'none')
    END INTO manager_role
    FROM public.boards b
    LEFT JOIN public.board_members bm ON b.id = bm.board_id AND bm.user_id = manager_uuid
    WHERE b.id = board_uuid;

    -- Owner can manage everyone
    IF manager_role = 'owner' THEN
        RETURN TRUE;
    END IF;

    -- Admin can manage collaborators and viewers
    IF manager_role = 'admin' AND target_role IN ('collaborator', 'viewer') THEN
        RETURN TRUE;
    END IF;

    RETURN FALSE;
END;
$$;

-- Get user's role on board
CREATE OR REPLACE FUNCTION public.get_user_board_role(board_uuid UUID, user_uuid UUID)
RETURNS TEXT
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
DECLARE
    user_role TEXT;
BEGIN
    SELECT CASE
        WHEN b.owner_id = user_uuid THEN 'owner'
        ELSE COALESCE(bm.role, 'none')
    END INTO user_role
    FROM public.boards b
    LEFT JOIN public.board_members bm ON b.id = bm.board_id AND bm.user_id = user_uuid
    WHERE b.id = board_uuid;

    RETURN COALESCE(user_role, 'none');
END;
$$;

--------------------------------------------------------------------
-- 7. SECURE FUNCTIONS FOR APPROVAL WORKFLOW
--------------------------------------------------------------------

-- Apply approved edit securely
CREATE OR REPLACE FUNCTION public.apply_proposed_edit(p_edit_id UUID)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
DECLARE
    v_edit RECORD;
    v_result JSONB := '{"success": false}';
BEGIN
    -- Load and lock the proposed edit
    SELECT * INTO v_edit
    FROM public.proposed_edits
    WHERE id = p_edit_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RETURN jsonb_build_object('success', false, 'error', 'Edit not found');
    END IF;

    IF v_edit.status <> 'approved' THEN
        RETURN jsonb_build_object('success', false, 'error', 'Edit not approved');
    END IF;

    -- Apply changes based on resource type and operation
    IF v_edit.resource_type = 'task' THEN
        IF v_edit.operation_type = 'create' THEN
            INSERT INTO public.tasks (
                id, title, description, column_id, board_id,
                priority, position, created_at, updated_at
            ) VALUES (
                COALESCE((v_edit.payload ->> 'id')::UUID, uuid_generate_v4()),
                v_edit.payload ->> 'title',
                v_edit.payload ->> 'description',
                (v_edit.payload ->> 'column_id')::UUID,
                v_edit.board_id,
                COALESCE(v_edit.payload ->> 'priority', 'Medium'),
                COALESCE((v_edit.payload ->> 'position')::INTEGER, 0),
                NOW(),
                NOW()
            );

        ELSIF v_edit.operation_type = 'update' THEN
            UPDATE public.tasks
            SET
                title = COALESCE(v_edit.payload ->> 'title', title),
                description = COALESCE(v_edit.payload ->> 'description', description),
                priority = COALESCE(v_edit.payload ->> 'priority', priority),
                assigned_to = CASE
                    WHEN v_edit.payload ? 'assigned_to' THEN
                        CASE WHEN v_edit.payload ->> 'assigned_to' = 'null'
                             THEN NULL
                             ELSE (v_edit.payload ->> 'assigned_to')::UUID
                        END
                    ELSE assigned_to
                END,
                deadline = CASE
                    WHEN v_edit.payload ? 'deadline' THEN
                        CASE WHEN v_edit.payload ->> 'deadline' = 'null'
                             THEN NULL
                             ELSE (v_edit.payload ->> 'deadline')::TIMESTAMPTZ
                        END
                    ELSE deadline
                END,
                completed = COALESCE((v_edit.payload ->> 'completed')::BOOLEAN, completed),
                updated_at = NOW(),
                version = version + 1
            WHERE id = v_edit.resource_id;

        ELSIF v_edit.operation_type = 'delete' THEN
            DELETE FROM public.tasks WHERE id = v_edit.resource_id;

        ELSIF v_edit.operation_type = 'move' THEN
            UPDATE public.tasks
            SET
                column_id = (v_edit.payload ->> 'column_id')::UUID,
                position = (v_edit.payload ->> 'position')::INTEGER,
                updated_at = NOW(),
                version = version + 1
            WHERE id = v_edit.resource_id;
        END IF;

    ELSIF v_edit.resource_type = 'column' THEN
        IF v_edit.operation_type = 'create' THEN
            INSERT INTO public.columns (
                id, board_id, title, position, created_at, updated_at
            ) VALUES (
                COALESCE((v_edit.payload ->> 'id')::UUID, uuid_generate_v4()),
                v_edit.board_id,
                v_edit.payload ->> 'title',
                COALESCE((v_edit.payload ->> 'position')::INTEGER, 0),
                NOW(),
                NOW()
            );

        ELSIF v_edit.operation_type = 'update' THEN
            UPDATE public.columns
            SET
                title = COALESCE(v_edit.payload ->> 'title', title),
                updated_at = NOW()
            WHERE id = v_edit.resource_id;

        ELSIF v_edit.operation_type = 'delete' THEN
            DELETE FROM public.columns WHERE id = v_edit.resource_id;
        END IF;

    ELSIF v_edit.resource_type = 'board' THEN
        IF v_edit.operation_type = 'update' THEN
            UPDATE public.boards
            SET
                title = COALESCE(v_edit.payload ->> 'title', title),
                description = COALESCE(v_edit.payload ->> 'description', description),
                updated_at = NOW(),
                version = version + 1
            WHERE id = v_edit.resource_id;
        END IF;
    END IF;

    -- Mark edit as applied
    UPDATE public.proposed_edits
    SET
        status = 'applied',
        updated_at = NOW()
    WHERE id = p_edit_id;

    -- Log the application
    INSERT INTO public.activity_log (
        user_id, board_id, action, description, metadata, created_at
    ) VALUES (
        v_edit.reviewer_id,
        v_edit.board_id,
        'edit_applied',
        format('Applied %s %s by %s', v_edit.operation_type, v_edit.resource_type,
               (SELECT name FROM public.users WHERE id = v_edit.proposed_by)),
        jsonb_build_object(
            'edit_id', p_edit_id,
            'resource_type', v_edit.resource_type,
            'operation_type', v_edit.operation_type,
            'proposed_by', v_edit.proposed_by
        ),
        NOW()
    );

    RETURN jsonb_build_object('success', true, 'edit_id', p_edit_id);
END;
$$;

-- Clean up expired pending edits
CREATE OR REPLACE FUNCTION public.cleanup_expired_edits()
RETURNS INTEGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
DECLARE
    cleanup_count INTEGER;
BEGIN
    UPDATE public.proposed_edits
    SET
        status = 'rejected',
        review_reason = 'Automatically rejected due to expiration',
        reviewed_at = NOW(),
        updated_at = NOW()
    WHERE status = 'pending'
      AND expires_at < NOW();

    GET DIAGNOSTICS cleanup_count = ROW_COUNT;
    RETURN cleanup_count;
END;
$$;


--------------------------------------------------------------------
-- 8. RLS POLICIES
--------------------------------------------------------------------

-- USERS
CREATE POLICY "All users can read users"
    ON users FOR SELECT TO authenticated USING (TRUE);

CREATE POLICY "Users can update own profile"
    ON users FOR UPDATE TO authenticated USING ((select auth.uid()) = id);

-- OTP TOKENS
CREATE POLICY "Users manage own OTP tokens"
    ON otp_tokens FOR ALL TO authenticated USING (
        email IN (SELECT email FROM users WHERE id = (select auth.uid()))
    );

-- BOARDS
CREATE POLICY "Members can view boards"
    ON boards FOR SELECT TO authenticated USING (
        owner_id = (select auth.uid())
        OR EXISTS (SELECT 1 FROM board_members WHERE board_id = id AND user_id = (select auth.uid()))
        OR is_public = TRUE
    );

CREATE POLICY "Owners can update boards directly"
    ON boards FOR UPDATE TO authenticated USING (owner_id = (select auth.uid()));

CREATE POLICY "Owners can delete boards"
    ON boards FOR DELETE TO authenticated USING (owner_id = (select auth.uid()));

CREATE POLICY "Authenticated users can create boards"
    ON boards FOR INSERT TO authenticated WITH CHECK (owner_id = (select auth.uid()));

-- BOARD MEMBERS
CREATE POLICY "Members can manage board members"
    ON board_members FOR ALL TO authenticated USING (
        user_has_board_access(board_id, (select auth.uid()))
        OR user_can_manage_member(board_id, (select auth.uid()), role)
    );

-- COLUMNS
CREATE POLICY "Members can manage columns"
    ON columns FOR ALL TO authenticated USING (
        user_has_board_access(board_id, (select auth.uid()))
        OR user_can_modify_directly(board_id, (select auth.uid()))
    );

-- TASKS
CREATE POLICY "Members can manage tasks"
    ON tasks FOR ALL TO authenticated USING (
        user_has_board_access(board_id, (select auth.uid()))
        OR user_can_modify_directly(board_id, (select auth.uid()))
    );

-- PROPOSED EDITS
CREATE POLICY "Users can manage proposed edits"
    ON proposed_edits FOR ALL TO authenticated USING (
        user_has_board_access(board_id, (select auth.uid()))
        OR user_can_approve_edits(board_id, (select auth.uid()))
    ) WITH CHECK (
        proposed_by = (select auth.uid())
        AND user_has_board_access(board_id, (select auth.uid()))
        AND get_user_board_role(board_id, (select auth.uid())) IN ('collaborator', 'admin', 'owner')
    );

-- APPROVAL NOTIFICATIONS
CREATE POLICY "Users can manage their notifications"
    ON approval_notifications FOR ALL TO authenticated USING (
        recipient_id = (select auth.uid())
    );

-- REALTIME SESSIONS
CREATE POLICY "Users manage own realtime sessions"
    ON realtime_sessions FOR ALL TO authenticated USING (user_id = (select auth.uid()));

-- USER PRESENCE
CREATE POLICY "Members can manage presence"
    ON user_presence FOR ALL TO authenticated USING (
        user_has_board_access(board_id, (select auth.uid()))
        OR user_id = (select auth.uid())
    );

-- ACTIVITY LOG
CREATE POLICY "Members can view activity log"
    ON activity_log FOR SELECT TO authenticated USING (
        user_has_board_access(board_id, (select auth.uid()))
    );

CREATE POLICY "System can insert activity log"
    ON activity_log FOR INSERT TO authenticated WITH CHECK (TRUE);

-- COMMENTS
CREATE POLICY "Members can manage comments"
    ON comments FOR ALL TO authenticated USING (
        EXISTS (
            SELECT 1
            FROM tasks t
            WHERE t.id = task_id
              AND user_has_board_access(t.board_id, (select auth.uid()))
        )
        OR user_id = (select auth.uid())
    ) WITH CHECK (
        user_id = (select auth.uid())
        AND EXISTS (
            SELECT 1
            FROM tasks t
            WHERE t.id = task_id
              AND user_has_board_access(t.board_id, (select auth.uid()))
              AND get_user_board_role(t.board_id, (select auth.uid())) IN ('owner','admin','collaborator')
        )
    );


--------------------------------------------------------------------
-- 9. TRIGGERS
--------------------------------------------------------------------

-- Auto‑update `updated_at` columns
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_boards_updated_at
    BEFORE UPDATE ON boards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_columns_updated_at
    BEFORE UPDATE ON columns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_proposed_edits_updated_at
    BEFORE UPDATE ON proposed_edits
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_comments_updated_at
    BEFORE UPDATE ON comments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Ensure board owner is automatically a board member with owner role
CREATE OR REPLACE FUNCTION ensure_board_owner_membership()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
BEGIN
    INSERT INTO public.board_members (board_id, user_id, role, joined_at)
    VALUES (NEW.id, NEW.owner_id, 'owner', NEW.created_at)
    ON CONFLICT (board_id, user_id)
    DO UPDATE SET role = 'owner';

    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_ensure_board_owner_membership
    AFTER INSERT ON boards
    FOR EACH ROW EXECUTE FUNCTION ensure_board_owner_membership();

-- Create notifications for pending approval requests
CREATE OR REPLACE FUNCTION create_approval_notifications()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
DECLARE
    approver_id UUID;
BEGIN
    -- Only create notifications for new pending edits
    IF NEW.status = 'pending' THEN
        -- Get all owners and admins who can approve
        FOR approver_id IN
            SELECT DISTINCT CASE
                WHEN b.owner_id IS NOT NULL THEN b.owner_id
                ELSE bm.user_id
            END
            FROM public.boards b
            LEFT JOIN public.board_members bm ON b.id = bm.board_id
                AND bm.role = 'admin'
            WHERE b.id = NEW.board_id
              AND (b.owner_id IS NOT NULL OR bm.user_id IS NOT NULL)
        LOOP
            INSERT INTO public.approval_notifications (proposed_edit_id, recipient_id)
            VALUES (NEW.id, approver_id)
            ON CONFLICT (proposed_edit_id, recipient_id) DO NOTHING;
        END LOOP;
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_create_approval_notifications
    AFTER INSERT ON proposed_edits
    FOR EACH ROW EXECUTE FUNCTION create_approval_notifications();


--------------------------------------------------------------------
-- 10. SCHEDULED CLEANUP (to be run periodically)
--------------------------------------------------------------------

-- Function to be called by a cron job or scheduler
-- Cleans up expired edits and old notifications
CREATE OR REPLACE FUNCTION public.maintenance_cleanup()
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
DECLARE
    expired_edits INTEGER;
    old_notifications INTEGER;
BEGIN
    -- Clean up expired edits
    expired_edits := public.cleanup_expired_edits();

    -- Clean up old read notifications (older than 30 days)
    DELETE FROM public.approval_notifications
    WHERE read_at IS NOT NULL
      AND read_at < NOW() - INTERVAL '30 days';
    GET DIAGNOSTICS old_notifications = ROW_COUNT;

    RETURN jsonb_build_object(
        'expired_edits_cleaned', expired_edits,
        'old_notifications_cleaned', old_notifications,
        'cleanup_time', NOW()
    );
END;
$$;


--------------------------------------------------------------------
-- 11. SECURITY GRANTS
--------------------------------------------------------------------

-- Revoke public execute on security definer functions
REVOKE EXECUTE ON FUNCTION public.apply_proposed_edit(UUID) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.cleanup_expired_edits() FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.maintenance_cleanup() FROM PUBLIC;

-- Grant execute to authenticated users only
-- Note: In Supabase, you might need to adjust these grants based on your auth setup
-- GRANT EXECUTE ON FUNCTION public.apply_proposed_edit(UUID) TO authenticated;
-- GRANT EXECUTE ON FUNCTION public.cleanup_expired_edits() TO service_role;
-- GRANT EXECUTE ON FUNCTION public.maintenance_cleanup() TO service_role;

--------------------------------------------------------------------
-- 12. SAMPLE DATA VIEWS (for testing/monitoring)
--------------------------------------------------------------------

-- View to see pending approvals by board (with proper security settings)
CREATE OR REPLACE VIEW pending_approvals_summary
WITH (security_invoker = on, security_barrier = true) AS
SELECT
    b.title as board_title,
    pe.board_id,
    COUNT(*) as pending_count,
    MIN(pe.created_at) as oldest_pending,
    MAX(pe.created_at) as newest_pending
FROM proposed_edits pe
JOIN boards b ON pe.board_id = b.id
WHERE pe.status = 'pending'
  AND pe.expires_at > NOW()
GROUP BY b.title, pe.board_id
ORDER BY pending_count DESC;

-- View to see user permissions by board (with proper security settings)
-- This view shows only boards/users the current user has access to
CREATE OR REPLACE VIEW user_board_permissions
WITH (security_invoker = on, security_barrier = true) AS
SELECT
    u.name as user_name,
    u.email as user_email,
    b.title as board_title,
    CASE
        WHEN b.owner_id = u.id THEN 'owner'
        ELSE COALESCE(bm.role, 'none')
    END as role,
    CASE
        WHEN b.owner_id = u.id OR
             (bm.role = 'admin') THEN 'direct'
        WHEN bm.role = 'collaborator' THEN 'approval_required'
        WHEN bm.role = 'viewer' THEN 'read_only'
        ELSE 'no_access'
    END as access_type
FROM users u
CROSS JOIN boards b
LEFT JOIN board_members bm ON b.id = bm.board_id AND u.id = bm.user_id
WHERE b.archived = FALSE
  -- Only show boards the current user has access to
  AND (
    b.owner_id = (select auth.uid())
    OR EXISTS (
      SELECT 1 FROM board_members bm_check
      WHERE bm_check.board_id = b.id
        AND bm_check.user_id = (select auth.uid())
    )
  )
ORDER BY b.title, role DESC;


--------------------------------------------------------------------
-- 13. MULTIPLE TASK ASSIGNEES FEATURE
-- Date: 2025-01-15
-- Description: Adds support for multiple assignees per task with individual completion status
--------------------------------------------------------------------

-- Create the task_assignees junction table
CREATE TABLE IF NOT EXISTS task_assignees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    completed BOOLEAN NOT NULL DEFAULT false,
    completed_at TIMESTAMPTZ,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(task_id, user_id)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_task_assignees_task_id ON task_assignees(task_id);
CREATE INDEX IF NOT EXISTS idx_task_assignees_user_id ON task_assignees(user_id);
CREATE INDEX IF NOT EXISTS idx_task_assignees_completed ON task_assignees(completed);
CREATE INDEX IF NOT EXISTS idx_task_assignees_task_user ON task_assignees(task_id, user_id);

-- Enable RLS
ALTER TABLE task_assignees ENABLE ROW LEVEL SECURITY;

-- RLS Policies (One policy per action to avoid multiple permissive policies warning)
CREATE POLICY "task_assignees_select_policy"
ON task_assignees FOR SELECT TO authenticated
USING (
    EXISTS (
        SELECT 1 FROM tasks t
        INNER JOIN boards b ON t.board_id = b.id
        LEFT JOIN board_members bm ON b.id = bm.board_id
        WHERE t.id = task_assignees.task_id
        AND (b.owner_id = (select auth.uid()) OR bm.user_id = (select auth.uid()))
    )
);

CREATE POLICY "task_assignees_insert_policy"
ON task_assignees FOR INSERT TO authenticated
WITH CHECK (
    EXISTS (
        SELECT 1 FROM tasks t
        INNER JOIN boards b ON t.board_id = b.id
        LEFT JOIN board_members bm ON b.id = bm.board_id
        WHERE t.id = task_assignees.task_id
        AND (
            b.owner_id = (select auth.uid())
            OR (bm.user_id = (select auth.uid()) AND bm.role IN ('owner', 'admin'))
        )
    )
);

CREATE POLICY "task_assignees_update_policy"
ON task_assignees FOR UPDATE TO authenticated
USING (
    -- Assignees can update their own assignment OR owners/admins can update any
    user_id = (select auth.uid())
    OR EXISTS (
        SELECT 1 FROM tasks t
        INNER JOIN boards b ON t.board_id = b.id
        LEFT JOIN board_members bm ON b.id = bm.board_id
        WHERE t.id = task_assignees.task_id
        AND (
            b.owner_id = (select auth.uid())
            OR (bm.user_id = (select auth.uid()) AND bm.role IN ('owner', 'admin'))
        )
    )
)
WITH CHECK (
    -- Assignees can only update their own assignment OR owners/admins can update any
    user_id = (select auth.uid())
    OR EXISTS (
        SELECT 1 FROM tasks t
        INNER JOIN boards b ON t.board_id = b.id
        LEFT JOIN board_members bm ON b.id = bm.board_id
        WHERE t.id = task_assignees.task_id
        AND (
            b.owner_id = (select auth.uid())
            OR (bm.user_id = (select auth.uid()) AND bm.role IN ('owner', 'admin'))
        )
    )
);

CREATE POLICY "task_assignees_delete_policy"
ON task_assignees FOR DELETE TO authenticated
USING (
    EXISTS (
        SELECT 1 FROM tasks t
        INNER JOIN boards b ON t.board_id = b.id
        LEFT JOIN board_members bm ON b.id = bm.board_id
        WHERE t.id = task_assignees.task_id
        AND (
            b.owner_id = (select auth.uid())
            OR (bm.user_id = (select auth.uid()) AND bm.role IN ('owner', 'admin'))
        )
    )
);

-- Migrate existing task assignments
INSERT INTO task_assignees (task_id, user_id, completed, completed_at, assigned_at)
SELECT
    t.id as task_id,
    t.assigned_to as user_id,
    t.completed,
    t.completed_at,
    t.created_at as assigned_at
FROM tasks t
WHERE t.assigned_to IS NOT NULL
ON CONFLICT (task_id, user_id) DO NOTHING;

-- Function to automatically update task completion status
CREATE OR REPLACE FUNCTION update_task_completion_status()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
BEGIN
    UPDATE public.tasks
    SET
        completed = (
            SELECT COUNT(*) = COUNT(*) FILTER (WHERE completed = true)
            FROM public.task_assignees
            WHERE task_id = COALESCE(NEW.task_id, OLD.task_id)
            HAVING COUNT(*) > 0
        ),
        completed_at = CASE
            WHEN (
                SELECT COUNT(*) = COUNT(*) FILTER (WHERE completed = true)
                FROM public.task_assignees
                WHERE task_id = COALESCE(NEW.task_id, OLD.task_id)
            ) THEN NOW()
            ELSE NULL
        END,
        updated_at = NOW()
    WHERE id = COALESCE(NEW.task_id, OLD.task_id);

    RETURN COALESCE(NEW, OLD);
END;
$$;

-- Trigger to auto-update task completion
DROP TRIGGER IF EXISTS trigger_update_task_completion ON task_assignees;
CREATE TRIGGER trigger_update_task_completion
AFTER INSERT OR UPDATE OR DELETE ON task_assignees
FOR EACH ROW
EXECUTE FUNCTION update_task_completion_status();

-- Helper functions
CREATE OR REPLACE FUNCTION get_task_assignees_count(task_uuid UUID)
RETURNS INTEGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM task_assignees WHERE task_id = task_uuid);
END;
$$;

CREATE OR REPLACE FUNCTION get_completed_assignees_count(task_uuid UUID)
RETURNS INTEGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM task_assignees WHERE task_id = task_uuid AND completed = true);
END;
$$;

-- View for easy querying
CREATE OR REPLACE VIEW task_assignees_with_users
WITH (security_invoker = on, security_barrier = true) AS
SELECT
    ta.id,
    ta.task_id,
    ta.user_id,
    ta.completed,
    ta.completed_at,
    ta.assigned_at,
    ta.assigned_by,
    u.name as user_name,
    u.email as user_email,
    u.avatar_url as user_avatar,
    t.title as task_title,
    t.board_id
FROM task_assignees ta
INNER JOIN users u ON ta.user_id = u.id
INNER JOIN tasks t ON ta.task_id = t.id;

-- Add comments to deprecated columns
COMMENT ON COLUMN tasks.assigned_to IS 'DEPRECATED: Use task_assignees table instead. Kept for backward compatibility.';
COMMENT ON COLUMN tasks.completed IS 'Auto-updated based on task_assignees completion status';

-- Trigger for updated_at
CREATE TRIGGER trg_task_assignees_updated_at
    BEFORE UPDATE ON task_assignees
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();