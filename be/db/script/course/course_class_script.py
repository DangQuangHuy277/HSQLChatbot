import os

import pdfplumber
import psycopg2
from dotenv import load_dotenv

from db.script.professor.script import extract_academic_rank_and_degree
from db.script.professor.advisor_script import get_class_maps

load_dotenv()
DATABASE_URL = os.getenv('DATABASE_URL')

def standardize_administrative_class_name(name):
    try:
        year = int(name[1:3]) + 1955
        suffix = name[3:]
        return f"QH-{year}-I/CQ-{suffix}"
    except (ValueError, IndexError):
        raise ValueError("Invalid format for administrative class name")



def upsert_course_class(cursor, semester_id, course_code, course_class_code, suggested_class, professor_name):
    # 1. Get or insert professor
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

    # 2. Get or insert course (assuming a course table exists with at least code and id)
    cursor.execute("SELECT id FROM course WHERE code = %s", (course_code,))
    course_row = cursor.fetchone()
    if course_row:
        course_id = course_row[0]
    else:
        print(f"Course {course_code} not found in the database.")
        return

    # 3. Fetch the administrative class id
    suggested_class_id = None
    suggested_class_name = standardize_administrative_class_name(suggested_class)
    cursor.execute("""
        SELECT id FROM administrative_class WHERE name = %s
    """, (suggested_class_name,))
    administrative_class_row = cursor.fetchone()
    if administrative_class_row:
        suggested_class_id = administrative_class_row[0]

    # 4. Upsert the course_class row.
    # The UNIQUE constraint on (code, semester_id) allows us to use ON CONFLICT.
    cursor.execute("""
        INSERT INTO course_class (code, course_id, semester_id, professor_id, suggested_class_id)
        VALUES (%s, %s, %s, %s, %s)
        ON CONFLICT (code, semester_id)
        DO UPDATE SET professor_id = EXCLUDED.professor_id,
                      suggested_class_id = EXCLUDED.suggested_class_id
        RETURNING id
    """, (course_class_code, course_id, semester_id, professor_id, suggested_class_id))
    course_class_id = cursor.fetchone()[0]
    return course_class_id


def insert_new_course_schedule(semester_id, suggested_class, course_code, course_class_code, course_name, credit,
                               capacity,
                               professor_name, day_of_week, period, location, group_identifier):
    with psycopg2.connect(DATABASE_URL) as conn:
        with conn.cursor() as cursor:
            # 1. Upsert the course_class and get its id.
            course_class_id = upsert_course_class(
                cursor,
                semester_id,
                course_code,
                course_class_code,
                suggested_class,
                professor_name
            )

            # Determine the session type.
            # For example, if the group_identifier is 'CL' (ignoring case and whitespace), we treat it as a theory session.
            session_type = 'Lý thuyết' if group_identifier.strip().upper() == 'CL' else 'Thực hành'


            # 4. Insert the course_class_schedule record.
            cursor.execute("""
                INSERT INTO course_class_schedule (course_class_id, day_of_week, lesson_range, session_type, group_identifier, location)
                VALUES (%s, %s, %s, %s, %s, %s)
            """, (course_class_id, day_of_week, period, session_type, group_identifier, location))
            conn.commit()


def insert_data_from_folder(folder_path, semester_id):
    for filename in os.listdir(folder_path):
        if not filename.endswith(".pdf"):
            continue
        pdf_path = os.path.join(folder_path, filename)
        with pdfplumber.open(pdf_path) as pdf:
            for page in pdf.pages:
                table = page.extract_table()
                if not table:
                    continue
                for row in table:
                    # Expecting at least 11 columns.
                    if len(row) < 11:
                        continue

                    (suggested_class, course_code, course_class_code, course_name, credit, capacity,
                     professor_name, day_of_week, period_str, location, group_identifier) = row

                    # Skip rows with missing required fields.
                    if not all([course_code, course_name, credit, professor_name, day_of_week, period_str, location]):
                        continue

                    try:
                        credit = int(credit)
                        # period_str might be something like "3-4" or "7" so we convert it into a list of ints.
                        if '-' in period_str:
                            period = [int(p) for p in period_str.split('-')]
                        else:
                            period = [int(period_str)]
                        insert_new_course_schedule(
                            semester_id,
                            suggested_class,
                            course_code,
                            course_class_code,
                            course_name,
                            credit,
                            capacity,
                            professor_name,
                            day_of_week,
                            period,
                            location,
                            group_identifier
                        )
                    except ValueError:
                        # Skip rows where conversion to int fails.
                        continue


# Run the script
if __name__ == "__main__":
    folder = "/home/huy/Code/Personal/KLTN/be/db/script/course/source"  # Update the path as needed.
    semester = '2024-2025-1'  # Replace with the actual semester identifier if needed.
    insert_data_from_folder(folder, semester)
