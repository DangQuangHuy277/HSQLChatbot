import os
import re

import pdfplumber
import psycopg2
from dotenv import load_dotenv

from db.script.professor.script import extract_academic_rank_and_degree

load_dotenv()
DATABASE_URL = os.getenv('DATABASE_URL')



def upsert_course_class(cursor, semester_id, course_id, course_class_code):
    cursor.execute("""
        SELECT id FROM course_class WHERE semester_id = %s AND course_id = %s AND code = %s;
    """, (semester_id, course_id, course_class_code))
    course_class_row = cursor.fetchone()
    if course_class_row:
        course_class_id = course_class_row[0]
    else:
        cursor.execute("""
            INSERT INTO course_class (code, course_id, semester_id)
                VALUES (%s, %s, %s)
            ON CONFLICT (code, semester_id) DO UPDATE SET code = EXCLUDED.code
            RETURNING id
        """, (course_class_code, course_id, semester_id))
        course_class_id = cursor.fetchone()[0]
        cursor.execute("""
                SELECT setval('course_class_id_seq', MAX(id), true) FROM course_class;""")

    return course_class_id


def upsert_course(cursor, course_code, course_name, credit):
    cursor.execute("SELECT id FROM course WHERE code = %s", (course_code,))
    course_row = cursor.fetchone()
    if course_row:
        course_id = course_row[0]
    else:
        # TODO: Handle the case where the course is not found in the database. Like: PES, etc.\
        cursor.execute(
            """
            INSERT INTO course (code, name, credits) VALUES (%s, %s, %s)
            ON CONFLICT (code) DO NOTHING
            RETURNING id;
            """, (course_code, course_name, credit)
        )
        course_id = cursor.fetchone()[0]
        cursor.execute("""
                SELECT setval('course_class_id_seq', MAX(id), true) FROM course_class;""")
    return course_id


def upsert_professors(cursor, professor_names):
    professor_ids = []
    for professor_name in re.split(r'\+|\n', professor_names):
        professor_name = professor_name.strip()
        academic_rank, degree, professor_name = extract_academic_rank_and_degree(professor_name)
        cursor.execute("SELECT id FROM professor WHERE name = %s", (professor_name,))
        professor_row = cursor.fetchone()
        if professor_row:
            professor_id = professor_row[0]
        else:
            cursor.execute(
                """
                INSERT INTO professor (name, academic_rank, degree) VALUES (%s, %s, %s)
                ON CONFLICT (name) DO NOTHING
                RETURNING id;
                """,
                (professor_name, academic_rank, degree))
            professor_id = cursor.fetchone()[0]
            cursor.execute("""
                SELECT setval('course_class_id_seq', MAX(id), true) FROM course_class;""")
        professor_ids.append(professor_id)
    return professor_ids


def insert_new_course_schedule(semester_id, course_code, course_class_code, course_name, credit,
                               capacity,
                               professor_name, day_of_week, period, location, group_identifier):
    with psycopg2.connect(DATABASE_URL) as conn:
        with conn.cursor() as cursor:
            # 1. Get or insert professor (assuming a professor table exists with at least name and id)
            professor_ids = upsert_professors(cursor, professor_name)
            if not professor_ids:
                print(f"Professor {professor_name} not found in the database.")
                return

            # 2. Get or insert course (assuming a course table exists with at least code and id)
            course_id = upsert_course(cursor, course_code, course_name, credit)
            if not course_id:
                print(f"Course {course_code} not found in the database.")
                return

            # 3. Upsert the course_class and get its id.
            course_class_id = upsert_course_class(
                cursor,
                semester_id,
                course_id,
                course_class_code
            )

            if not course_class_id:
                print(f"Course class {course_class_code} not found in the database.")
                return

            # 4. Insert the course_class_schedule record.
            # Determine the session type.
            # For example, if the group_identifier is 'CL' (ignoring case and whitespace), we treat it as a theory session.
            if group_identifier:
                session_type = 'Lý thuyết' if group_identifier.strip().upper() == 'CL' or group_identifier.strip().upper() == 'LT' else 'Thực hành'
            else:
                session_type = ''
            cursor.execute("""
                INSERT INTO course_class_schedule (course_class_id, day_of_week, lesson_range, session_type, group_identifier, location)
                VALUES (%s, %s, %s, %s, %s, %s)
                RETURNING id;
            """, (course_class_id, day_of_week, period, session_type, group_identifier, location))
            course_class_schedule_id = cursor.fetchone()[0]
            if not course_class_schedule_id:
                print(f"Failed to insert course class schedule for {course_class_code}.")
                return

            # 5. Insert the course_schedule_instructor
            for professor_id in professor_ids:
                cursor.execute("""
                    INSERT INTO course_schedule_instructor (course_class_schedule_id, professor_id)
                    VALUES (%s, %s)
                """, (course_class_schedule_id, professor_id))
            conn.commit()


def parse_course_row(row, semester):
    if semester == "2024-2025-1":
        # Unpack 2024-2025-1 format
        _, course_code, course_class_code, course_name, credit, capacity, \
            professor_name, day_of_week, period_str, location, group_identifier = row

    elif semester == "2024-2025-2":
        # Unpack 2024-2025-2 format
        course_class_code, course_code, course_name, _, credit, group_identifier, capacity, \
            professor_name, day_of_week, period_str, location = row

    elif semester in ["2023-2024-1", "2023-2024-2"]:
        # Unpack 2023-2024 format
        course_code, course_name, credit, course_class_code, capacity, \
            professor_name, day_of_week, period_str, location, group_identifier = row

    else:
        raise ValueError(f"Unsupported semester format: {semester}")

    return {
        'course_code': course_code,
        'course_class_code': course_class_code,
        'course_name': course_name,
        'credit': credit,
        'capacity': capacity,
        'professor_name': professor_name,
        'day_of_week': day_of_week,
        'period_str': period_str,
        'location': location,
        'group_identifier': group_identifier
    }


def insert_data_from_folder(folder_path):
    for filename in os.listdir(folder_path):
        if not filename.endswith(".pdf"):
            continue

        # Extract semester from filename (remove .pdf extension)
        semester_id = filename[:-4]  # Removes the last 4 characters ('.pdf')

        pdf_path = os.path.join(folder_path, filename)
        with pdfplumber.open(pdf_path) as pdf:
            for page in pdf.pages:
                table = page.extract_table()
                if not table:
                    continue

                for row in table:
                    # Check minimum required columns based on semester format
                    min_columns = 10 if semester_id in ["2023-2024-1", "2023-2024-2"] else 11
                    if len(row) < min_columns:
                        continue

                    try:
                        # Parse the row based on semester format
                        course_data = parse_course_row(row, semester_id)

                        # Skip rows with missing required fields
                        required_fields = [
                            course_data['course_code'],
                            course_data['course_name'],
                            course_data['credit'],
                            course_data['course_class_code'],
                            course_data['professor_name'],
                            course_data['day_of_week'],
                            course_data['period_str'],
                            course_data['location']
                        ]

                        if not all(required_fields):
                            continue

                        # Convert credit to integer
                        credit = int(course_data['credit'])

                        # Parse period string
                        period_str = course_data['period_str']
                        if '-' in period_str:
                            period = [int(p) for p in period_str.split('-')]
                        else:
                            period = [int(period_str)]

                        # Insert into database
                        insert_new_course_schedule(
                            semester_id,
                            course_data['course_code'],
                            course_data['course_class_code'],
                            course_data['course_name'],
                            credit,
                            course_data['capacity'],
                            course_data['professor_name'],
                            course_data['day_of_week'],
                            period,
                            course_data['location'],
                            course_data['group_identifier']
                        )

                    except (ValueError, KeyError) as e:
                        # Skip rows where parsing or conversion fails
                        continue


if __name__ == "__main__":
    folder = "/home/huy/Code/Personal/KLTN/be/db/script/course/source"
    insert_data_from_folder(folder)
