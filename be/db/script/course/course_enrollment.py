import os
from datetime import datetime

import psycopg2
import requests
from bs4 import BeautifulSoup
from dotenv import load_dotenv

load_dotenv()
DATABASE_URL = os.getenv('DATABASE_URL')


def crawl_and_store_data(conn, term_id, semester_id):
    page = 1
    total_records = 0
    cnt = 0
    while cnt < 5:
        cnt += 1
        url = f"http://112.137.129.87/qldt/index.php?SinhvienLmh%5Bterm_id%5D={term_id}&SinhvienLmh_page=500000&pageSize=10000&ajax=sinhvien-lmh-grid&r=sinhvienLmh%2Fadmin"
        try:
            print(f"Fetching page {page}...")
            response = requests.get(url)
            response.raise_for_status()

            soup = BeautifulSoup(response.content, "html.parser")
            table = soup.find("table")

            if not table or not table.find('tbody') or len(table.find('tbody').find_all('tr')) == 0:
                print(f"No more data found on page {page}. Stopping.")
                break

            rows = table.find('tbody').find_all('tr')
            batch_data = []

            for row in rows:
                columns = row.find_all('td')
                if len(columns) < 10:
                    continue  # Skip invalid rows

                try:
                    student_code = columns[1].text.strip()
                    student_name = columns[2].text.strip()
                    student_birthday_str = columns[3].text.strip()
                    student_administrative_class = columns[4].text.strip()
                    course_class_code = columns[5].text.strip()
                    course_name = columns[6].text.strip()
                    course_group_identifier = columns[7].text.strip()
                    course_credit = int(columns[8].text.strip()) if columns[8].text.strip().isdigit() else 0
                    enrollment_type = columns[9].text.strip()

                    # Parse birthday
                    try:
                        student_birthday = datetime.strptime(student_birthday_str, "%d/%m/%Y").date()
                    except ValueError:
                        student_birthday = None

                    batch_data.append({
                        "student_code": student_code,
                        "student_name": student_name,
                        "student_birthday": student_birthday,
                        "student_administrative_class": student_administrative_class,
                        "course_class_code": course_class_code,
                        "course_name": course_name,
                        "course_group_identifier": course_group_identifier,
                        "course_credit": course_credit,
                        "enrollment_type": enrollment_type,
                        "term_id": term_id
                    })
                except Exception as e:
                    print(f"Error processing row: {e}")
                    continue

            # Store batch data
            processed_records = store_data_in_db(conn, batch_data, semester_id)
            total_records += processed_records
            print(f"Processed {processed_records} records from page {page}")

            if len(rows) < 1000:  # Assuming we've reached the last page if less than expected records
                print("Reached last page with fewer records than maximum. Stopping.")
                break

            page += 1

        except requests.exceptions.RequestException as e:
            print(f"Error during request (page {page}): {e}")
            break
        except Exception as e:
            print(f"An error occurred (page {page}): {e}")
            break

    print(f"Total records processed: {total_records}")
    return total_records


def store_data_in_db(conn, data, semester_id):
    cursor = conn.cursor()
    processed_count = 0

    try:
        for item in data:
            # 1. Check if administrative class exists
            cursor.execute("SELECT id FROM administrative_class WHERE name = %s",
                           (item["student_administrative_class"],))
            administrative_class_id = cursor.fetchone()
            if not administrative_class_id:
                cursor.execute("""
                    INSERT INTO administrative_class (name)
                    VALUES (%s)
                    RETURNING id;
                """, (item["student_administrative_class"],))
                administrative_class_id = cursor.fetchone()[0]
            else:
                administrative_class_id = administrative_class_id[0]

            # 2. Check if the student exists
            cursor.execute("SELECT id FROM student WHERE code = %s", (item["student_code"],))
            student_id = cursor.fetchone()
            if not student_id:
                cursor.execute("""
                    INSERT INTO student (code, name, birthday, administrative_class_id)
                    VALUES (%s, %s, %s, %s)
                    RETURNING id;
                """, (item["student_code"], item["student_name"], item["student_birthday"], administrative_class_id))
                student_id = cursor.fetchone()[0]
            else:
                student_id = student_id[0]

            # 3. Check if the course exists
            course_code = item["course_class_code"].split(' ')[0]
            cursor.execute("SELECT id FROM course WHERE code = %s", (course_code,))
            course_id = cursor.fetchone()
            if not course_id:
                cursor.execute("""
                    INSERT INTO course (code, name, credits)
                    VALUES (%s, %s, %s)
                    RETURNING id;
                """, (course_code, item["course_name"], item["course_credit"]))
                course_id = cursor.fetchone()[0]
            else:
                course_id = course_id[0]

            # 4. Check if the course class exists
            cursor.execute("SELECT id FROM course_class WHERE code = %s", (item["course_class_code"],))
            course_class_id = cursor.fetchone()
            if not course_class_id:
                cursor.execute("""
                    INSERT INTO course_class (code, course_id, semester_id)
                    VALUES (%s, %s, %s)
                    RETURNING id;
                """, (item["course_class_code"], course_id, semester_id))
                course_class_id = cursor.fetchone()[0]
            else:
                course_class_id = course_class_id[0]

            # 5. Check if the course_class_schedule exists
            cursor.execute("""
                SELECT id FROM course_class_schedule
                WHERE course_class_id = %s AND group_identifier = %s
            """, (course_class_id, item["course_group_identifier"]))
            course_class_schedule_id = cursor.fetchone()
            if not course_class_schedule_id:
                cursor.execute("""
                    INSERT INTO course_class_schedule (course_class_id, group_identifier)
                    VALUES (%s, %s)
                    RETURNING id;
                """, (course_class_id, item["course_group_identifier"]))
                course_class_schedule_id = cursor.fetchone()[0]
            else:
                course_class_schedule_id = course_class_schedule_id[0]
            course_class_schedule_ids = [course_class_schedule_id]
            if item["course_group_identifier"] != "CL":
                cursor.execute("""
                    SELECT id FROM course_class_schedule
                    WHERE course_class_id = %s AND group_identifier = 'CL'
                """, (course_class_id,))
                course_class_schedule_id = cursor.fetchone()
                if not course_class_schedule_id:
                    cursor.execute("""
                        INSERT INTO course_class_schedule (course_class_id, group_identifier)
                        VALUES (%s, 'CL')
                        RETURNING id;
                    """, (course_class_id,))
                    course_class_schedule_id = cursor.fetchone()[0]
                else:
                    course_class_schedule_id = course_class_schedule_id[0]
                course_class_schedule_ids.append(course_class_schedule_id)

            # 6. Link student to course class
            cursor.execute("""
                INSERT INTO course_class_enrollment (student_id, course_class_id, enrollment_type)
                VALUES (%s, %s, %s)
                ON CONFLICT(student_id, course_class_id) DO UPDATE SET enrollment_type = EXCLUDED.enrollment_type
                RETURNING id;
            """, (student_id, course_class_id, item["enrollment_type"]))
            course_class_enrollment_id = cursor.fetchone()[0]

            # 7. Link student to course class schedule
            for course_class_schedule_id in course_class_schedule_ids:
                cursor.execute("""
                    INSERT INTO student_course_class_schedule (course_class_enrollment_id, course_class_schedule_id)
                    VALUES (%s, %s)
                    ON CONFLICT DO NOTHING;
                """, (course_class_enrollment_id, course_class_schedule_id))

        processed_count += 1

        conn.commit()
        return processed_count
    except Exception as e:
        conn.rollback()
        print(f"Database error: {e}")
        raise


if __name__ == "__main__":
    semester_map = {
        "042": "2024-2025-2",
        "041": "2024-2025-1",
        "039": "2023-2024-2",
        "038": "2023-2024-1",
    }
    try:
        with psycopg2.connect(DATABASE_URL) as conn:
            for term_id, semester_id in semester_map.items():
                print(f"Processing term {term_id}...")
                total_records = crawl_and_store_data(conn, term_id, semester_id)
            print(f"Crawling and data storage complete. Total records: {total_records}")
    except Exception as e:
        print(f"Script execution failed: {e}")
