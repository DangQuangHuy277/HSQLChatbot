\c university;

CREATE TABLE IF NOT EXISTS department
(
    id   SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS professor
(
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(100),
    email         VARCHAR(100),
    department_id INT
--     FOREIGN KEY (department_id) REFERENCES department (id)
);

CREATE TABLE IF NOT EXISTS program
(
    id                SERIAL PRIMARY KEY,
    code              VARCHAR(10) UNIQUE NOT NULL, -- Mã tuyển sinh
    name              VARCHAR(100)       NOT NULL, -- Tên ngành/chương trình đào tạo
    degree_type       VARCHAR(50)        NOT NULL, -- Loại bằng tốt nghiệp (e.g., 'Cử nhân', 'Kỹ sư')
    training_duration DECIMAL(3, 1)      NOT NULL, -- Thời gian đào tạo (e.g., 4.0, 4.5 years),
    abbreviation      VARCHAR(10)                  -- Tên viết tắt

);



CREATE TABLE IF NOT EXISTS administrative_class
(
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(100) UNIQUE NOT NULL,
    program_id INT,
    advisor_id INT
--     FOREIGN KEY (program_id) REFERENCES program (id),
--     FOREIGN KEY (advisor_id) REFERENCES professor (id)
);

CREATE TABLE IF NOT EXISTS semester
(
    id   SERIAL PRIMARY KEY,
    name VARCHAR(100)
);

CREATE TABLE IF NOT EXISTS course
(
    id               SERIAL PRIMARY KEY,
    code             VARCHAR(10),   -- Course code, e.g., 'CS101'
    name             VARCHAR(100),  -- Course name, e.g., 'Introduction to Programming'
    credits          INT,           -- Credit hours for the course
    program_id       INT,           -- Foreign key to the related program
    practice_hours   INT DEFAULT 0, -- Hours dedicated to practice sessions
    theory_hours     INT DEFAULT 0, -- Hours dedicated to theory sessions
    self_learn_hours INT DEFAULT 0  -- Hours dedicated to self-study
);

CREATE TABLE IF NOT EXISTS course_class
(
    id           SERIAL PRIMARY KEY,
    code         VARCHAR(10),
    course_id    INT,
    semester_id  INT,
    professor_id INT
--     FOREIGN KEY (course_id) REFERENCES course (id),
--     FOREIGN KEY (semester_id) REFERENCES semester (id),
--     FOREIGN KEY (professor_id) REFERENCES professor (id)
);

CREATE TABLE IF NOT EXISTS course_class_schedule
(
    id              SERIAL PRIMARY KEY,
    course_class_id INT,         -- Foreign key to course_class
    day_of_week     VARCHAR(20), -- Day of the week, e.g., 'Monday', 'Tuesday'
    start_period    INT,         -- Start period, e.g., 2 (for period 2)
    end_period      INT,         -- End period, e.g., 5 (for period 5)
    location        VARCHAR(50)  -- Location, e.g., 'Room P108'
--     FOREIGN KEY (course_class_id) REFERENCES course_class (id)
);

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

CREATE TABLE IF NOT EXISTS student
(
    id                      SERIAL PRIMARY KEY,
    code                    VARCHAR(10) UNIQUE NOT NULL, -- Student code/ID (e.g., "S12345678")
    name                    VARCHAR(100),                -- Student's full name
    gender                  VARCHAR(20),                 -- Gender (e.g., 'Male', 'Female', 'Non-binary')
    birthday                DATE,                        -- Birthday (e.g., '2000-01-15')
    email                   VARCHAR(100),                -- Email address
    program_id              INT,                         -- Foreign key to program
    administrative_class_id INT                          -- Foreign key to administrative_class
--     FOREIGN KEY (program_id) REFERENCES program (id),
--     FOREIGN KEY (administrative_class_id) REFERENCES administrative_class (id)
);

