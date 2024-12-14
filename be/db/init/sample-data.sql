
-- Insert sample data into departments
INSERT INTO departments (name, location)
VALUES ('Computer Science', 'Building A'),
       ('Mathematics', 'Building B'),
       ('Physics', 'Building C'),
       ('English', 'Building D');

-- Insert sample data into professors
INSERT INTO professors (name, email, department_id)
VALUES ('Dr. Alice Smith', 'alice.smith@university.edu', 1),
       ('Dr. Bob Johnson', 'bob.johnson@university.edu', 2),
       ('Dr. Carol White', 'carol.white@university.edu', 3),
       ('Dr. David Brown', 'david.brown@university.edu', 4);

-- Insert sample data into students
INSERT INTO students (name, email, enrollment_year, major_id)
VALUES ('John Doe', 'john.doe@university.edu', 2021, 1),
       ('Jane Roe', 'jane.roe@university.edu', 2022, 2),
       ('Sam Smith', 'sam.smith@university.edu', 2020, 3),
       ('Emily Davis', 'emily.davis@university.edu', 2023, 4);

-- Insert sample data into courses
INSERT INTO courses (name, code, department_id, professor_id, credits)
VALUES ('Introduction to Programming', 'CS101', 1, 1, 4),
       ('Data Structures', 'CS201', 1, 1, 3),
       ('Calculus I', 'MATH101', 2, 2, 4),
       ('Linear Algebra', 'MATH201', 2, 2, 3),
       ('Classical Mechanics', 'PHYS101', 3, 3, 4),
       ('English Literature', 'ENG101', 4, 4, 3);

-- Insert sample data into enrollments
INSERT INTO enrollments (student_id, course_id, grade)
VALUES (1, 1, 'A'),
       (1, 2, 'B'),
       (2, 3, 'A'),
       (2, 4, 'B'),
       (3, 5, 'C'),
       (4, 6, 'A');
