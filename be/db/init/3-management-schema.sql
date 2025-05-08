CREATE TABLE IF NOT EXISTS user_account
(
    id       SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL, -- Unique login identifier (e.g., student code or email)
    password VARCHAR(255)       NOT NULL  -- Hashed password (never store plain text)
);


-- Create Conversation table
CREATE TABLE Conversation (
                               id SERIAL PRIMARY KEY,
                               user_id INTEGER NOT NULL,
                               title VARCHAR(255),
                               disabled BOOLEAN DEFAULT FALSE,
                               created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                               updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                               deleted_at TIMESTAMP
);

-- Create Message table
CREATE TABLE Message (
                          id SERIAL PRIMARY KEY,
                          conversation_id INTEGER NOT NULL,
                          sender_type VARCHAR(10) NOT NULL CHECK (sender_type IN ('user', 'bot')),
                          content TEXT NOT NULL,
                          created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                          deleted_at TIMESTAMP
);

-- Add foreign key constraints
ALTER TABLE Conversation
    ADD CONSTRAINT fk_Conversation_user_id
        FOREIGN KEY (user_id)
            REFERENCES user_account(id)
            ON DELETE CASCADE;

ALTER TABLE Message
    ADD CONSTRAINT fk_Message_conversation_id
        FOREIGN KEY (conversation_id)
            REFERENCES Conversation(id)
            ON DELETE CASCADE;

-- Indexes for Conversation
CREATE INDEX idx_Conversation_user_id ON Conversation(user_id);
CREATE INDEX idx_Conversation_user_id_deleted_at ON Conversation(user_id, deleted_at);
CREATE INDEX idx_Conversation_user_id_disabled_deleted_at ON Conversation(user_id, disabled, deleted_at);
CREATE INDEX idx_Conversation_updated_at ON Conversation(updated_at);

-- Indexes for Message
CREATE INDEX idx_Message_conversation_id ON Message(conversation_id);
CREATE INDEX idx_Message_conversation_id_deleted_at ON Message(conversation_id, deleted_at);
CREATE INDEX idx_Message_created_at ON Message(created_at);




-- Public: all
CREATE TABLE IF NOT EXISTS program_semester_fee
(
    id             SERIAL PRIMARY KEY,
    program_id     INT,                                                       -- Foreign key to the program
    semester_id    INT,                                                       -- Foreign key to the semester
    fee_type       VARCHAR(20) CHECK ( fee_type IN ('PER_CREDIT', 'FIXED') ), -- 'PER_CREDIT' or 'FIXED'
    fee_per_credit NUMERIC(10, 2),                                            -- Fee per credit (nullable if fee_type = 'FIXED')
    fixed_fee      NUMERIC(10, 2)                                             -- Fixed fee for the semester (nullable if fee_type = 'PER_CREDIT')
--     FOREIGN KEY (program_id) REFERENCES program (id),
--     FOREIGN KEY (semester_id) REFERENCES semester (id)
);

CREATE TABLE IF NOT EXISTS scholarship
(
    id                   SERIAL PRIMARY KEY,                                                                  -- Unique ID for the scholarship
    name                 VARCHAR(100),                                                                        -- Scholarship name
    description          TEXT,                                                                                -- Description
    type                 VARCHAR(20) CHECK (type IN ('ACADEMIC', 'CORPORATE')),                               -- Scholarship type
    subtype              VARCHAR(20) CHECK (subtype IN ('EXCELLENT', 'GOOD')),                                -- Scholarship subtype (only for academic type)
    period_unit          VARCHAR(20) CHECK (period_unit IN ('SEMESTER', 'ACADEMIC_YEAR')) DEFAULT 'SEMESTER', -- Period unit
    amount               NUMERIC(10, 2),                                                                      -- Scholarship amount
    is_recurring         BOOLEAN                                                          DEFAULT FALSE,      -- Indicates whether the scholarship is recurring
    sponsor_name         VARCHAR(100),                                                                        -- Sponsor name (for corporate scholarships)
    eligibility_criteria TEXT                                                                                 -- Eligibility criteria (optional)
);

CREATE TABLE IF NOT EXISTS student_scholarship
(
    id             SERIAL PRIMARY KEY, -- Unique ID for the student-scholarship association
    student_id     INT,                -- Foreign key to the student table
    scholarship_id INT                 -- Foreign key to the scholarship table
--     FOREIGN KEY (student_id) REFERENCES student (id) ON DELETE CASCADE,        -- Cascade deletion when student is deleted
--     FOREIGN KEY (scholarship_id) REFERENCES scholarship (id) ON DELETE CASCADE -- Cascade deletion when scholarship is deleted
);