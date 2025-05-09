--Public: all
CREATE TABLE IF NOT EXISTS faculty
(
    id   SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    type VARCHAR(50)
);

--Each professor can only access their own data
CREATE TABLE IF NOT EXISTS professor
(
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(100) NOT NULL UNIQUE,
    email         VARCHAR(100),
    academic_rank VARCHAR(50), -- Học hàm (e.g., Professor)
    degree        VARCHAR(50), -- Học vị (e.g., PhD)
    department_id INT
--     FOREIGN KEY (department_id) REFERENCES department (id)
);


-- Public: all
CREATE TABLE IF NOT EXISTS program
(
    id                SERIAL PRIMARY KEY,
    code              VARCHAR(10) UNIQUE NOT NULL, -- Mã tuyển sinh
    name              VARCHAR(100)       NOT NULL, -- Tên ngành/chương trình đào tạo
    degree_type       VARCHAR(50)        NOT NULL, -- Loại bằng tốt nghiệp (e.g., 'Cử nhân', 'Kỹ sư')
    training_duration DECIMAL(3, 1)      NOT NULL, -- Thời gian đào tạo (e.g., 4.0, 4.5 years),
    abbreviation      VARCHAR(10)                  -- Tên viết tắt

);


-- Only advisors or student in this class can access his/her own class
CREATE TABLE IF NOT EXISTS administrative_class
(
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(100) UNIQUE NOT NULL,
    program_id INT,
    advisor_id INT
--     FOREIGN KEY (program_id) REFERENCES program (id),
--     FOREIGN KEY (advisor_id) REFERENCES professor (id)
);

-- Public: all
CREATE TABLE IF NOT EXISTS semester
(
    id   SERIAL PRIMARY KEY,
    name VARCHAR(100)
);

-- Public: all
CREATE TABLE IF NOT EXISTS course
(
    id               SERIAL PRIMARY KEY,
    code             VARCHAR(10) NOT NULL UNIQUE, -- Course code, e.g., 'CS101'
    name             VARCHAR(100),                -- Course name, e.g., 'Introduction to Programming'
    english_name     VARCHAR(100),                -- English name of the course
    credits          INT,                         -- Credit hours for the course
    practice_hours   INT DEFAULT 0,               -- Hours dedicated to practice sessions
    theory_hours     INT DEFAULT 0,               -- Hours dedicated to theory sessions
    self_learn_hours INT DEFAULT 0,               -- Hours dedicated to self-study
    prerequisite     INT                          -- Foreign key to the prerequisite course
);

-- Public: all
CREATE TABLE IF NOT EXISTS course_program -- Junction table for many-to-many relationship between course and program
(
    course_id  INT NOT NULL,            -- Foreign key to the course
    program_id INT NOT NULL,            -- Foreign key to the program
    PRIMARY KEY (course_id, program_id) -- Composite primary key
--     FOREIGN KEY (course_id) REFERENCES course(id),
--     FOREIGN KEY (program_id) REFERENCES program(id)
);

--Public:all
CREATE TABLE IF NOT EXISTS course_class
(
    id           SERIAL PRIMARY KEY,
    code         VARCHAR(10),
    course_id    INT         NOT NULL,
    semester_id  VARCHAR(11) NOT NULL,
    UNIQUE (code, semester_id)
-- FOREIGN KEY (course_id) REFERENCES course (id),
-- FOREIGN KEY (professor_id) REFERENCES professor (id)
);

-- Public: all
CREATE TABLE IF NOT EXISTS course_class_schedule
(
    id               SERIAL PRIMARY KEY,
    course_class_id  INT         NOT NULL,
    day_of_week      VARCHAR(10) NOT NULL, -- e.g. '2' for Monday or use full names if preferred
    lesson_range     VARCHAR(10),          -- e.g. '3-4' or '7-8' or '9-10'
    session_type     VARCHAR(20),          -- e.g. 'theory' or 'practice'
    group_identifier VARCHAR(20),          -- e.g. 'CL' for theory, '1', '2' for practice sessions
    location         VARCHAR(50) NOT NULL  -- e.g. '208-GĐ3', '214-GĐ3'
--     CONSTRAINT fk_course_class FOREIGN KEY (course_class_id)
--         REFERENCES course_class (id),
--     CONSTRAINT chk_session_group
--         CHECK (
--             (session_type = 'theory' AND (group_identifier = 'CL' OR group_identifier IS NULL))
--                 OR
--             (session_type = 'practice' AND group_identifier IS NOT NULL)
--             )
);

-- Public: all
create table course_schedule_instructor
(
    course_class_schedule_id integer,
    professor_id             integer,
    constraint course_schedule_instructor_pk
        primary key (course_class_schedule_id, professor_id)
);


-- Each student can only access their own data
CREATE TABLE IF NOT EXISTS student
(
    id                      SERIAL PRIMARY KEY,
    code                    VARCHAR(10) UNIQUE NOT NULL, -- Student code/ID (e.g., "S12345678")
    name                    VARCHAR(100),                -- Student's full name
    gender                  VARCHAR(20),                 -- Gender (e.g., 'Male', 'Female', 'Non-binary')
    birthday                DATE,                        -- Birthday (e.g., '2000-01-15')
    email                   VARCHAR(100),                -- Email address                       -- Foreign key to program
    administrative_class_id INT                           -- Foreign key to administrative_class
--     FOREIGN KEY (program_id) REFERENCES program (id),
--     FOREIGN KEY (administrative_class_id) REFERENCES administrative_class (id)
);

-- User can view data of students in the same class
CREATE TABLE IF NOT EXISTS course_class_enrollment
(
    id              SERIAL PRIMARY KEY,   -- Added id
    student_id      INT NOT NULL,
    course_class_id INT NOT NULL,
    enrollment_type VARCHAR(30),
    grade           VARCHAR(5),
    UNIQUE (student_id, course_class_id) -- Keep the unique constraint
--     FOREIGN KEY (student_id) REFERENCES student (id),
--     FOREIGN KEY (course_class_id) REFERENCES course_class (id)
);

-- User can view data of students in the same class
CREATE TABLE IF NOT EXISTS student_course_class_schedule
(
    id                       SERIAL PRIMARY KEY,                -- Added id
    course_class_enrollment_id  INT NOT NULL,
    course_class_schedule_id INT NOT NULL,
    UNIQUE (course_class_enrollment_id, course_class_schedule_id) -- Keep the unique constraint
--     FOREIGN KEY (course_class_enrollment_id) REFERENCES student_course_class (id),
--     FOREIGN KEY (course_class_schedule_id) REFERENCES course_class_schedule (id)
);

