-- Switch to the databasea
\c university;

-- Create the departments table
CREATE TABLE IF NOT EXISTS departments
(
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    location   VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create the professors table
CREATE TABLE IF NOT EXISTS professors
(
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(100)        NOT NULL,
    email         VARCHAR(100) UNIQUE NOT NULL,
    department_id INT                 REFERENCES departments (id) ON DELETE SET NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create the students table
CREATE TABLE IF NOT EXISTS students
(
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(100)        NOT NULL,
    email           VARCHAR(100) UNIQUE NOT NULL,
    enrollment_year INT                 NOT NULL,
    major_id        INT                 REFERENCES departments (id) ON DELETE SET NULL,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create the courses table
CREATE TABLE IF NOT EXISTS courses
(
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(100)       NOT NULL,
    code          VARCHAR(10) UNIQUE NOT NULL,
    department_id INT                REFERENCES departments (id) ON DELETE SET NULL,
    professor_id  INT                REFERENCES professors (id) ON DELETE SET NULL,
    credits       INT                NOT NULL CHECK (credits > 0),
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create the enrollments table
CREATE TABLE IF NOT EXISTS enrollments
(
    id              SERIAL PRIMARY KEY,
    student_id      INT REFERENCES students (id) ON DELETE CASCADE,
    course_id       INT REFERENCES courses (id) ON DELETE CASCADE,
    enrollment_date DATE DEFAULT CURRENT_DATE,
    grade           CHAR(2) CHECK (grade IN ('A', 'B', 'C', 'D', 'F', NULL))
);
