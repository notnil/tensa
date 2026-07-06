import os
import json

# Paths
BASE_DIR = "/home/logan/Documents/data/tensa-recordings"
INDEX_FILE = os.path.join(BASE_DIR, "localization_index.json")
# The new web_labeler uses dataset/timestamp_ns as key.
CENTRAL_LABELS_FILE = os.path.join(BASE_DIR, "localization_labels.jsonl")

def migrate_labels():
    if not os.path.exists(INDEX_FILE):
        print(f"Error: Index file {INDEX_FILE} not found.")
        return

    with open(INDEX_FILE, 'r') as f:
        sessions = json.load(f)

    all_labels = []
    found_sessions = 0
    total_records = 0

    for session in sessions:
        session_name = session['name']
        # Old labels were in session_dir/all_frames/labels.jsonl
        old_labels_path = os.path.join(BASE_DIR, session_name, "all_frames", "labels.jsonl")
        
        if os.path.exists(old_labels_path):
            print(f"Migrating labels from session: {session_name}...")
            session_count = 0
            with open(old_labels_path, 'r') as f:
                for line in f:
                    if not line.strip(): continue
                    try:
                        record = json.loads(line)
                        # The new backend expects 'dataset' and 'timestamp_ns' (as string usually)
                        record['dataset'] = session_name
                        # Ensure timestamp_ns is a string if it's not
                        if 'timestamp_ns' in record:
                            record['timestamp_ns'] = str(record['timestamp_ns'])
                        
                        all_labels.append(record)
                        session_count += 1
                        total_records += 1
                    except Exception as e:
                        print(f"  Error parsing line in {session_name}: {e}")
            
            print(f"  Found {session_count} records.")
            found_sessions += 1
        else:
            print(f"No existing labels found for session: {session_name}")

    if all_labels:
        # Sort by dataset and frame_idx if available
        all_labels.sort(key=lambda r: (r.get('dataset', ''), r.get('frame_idx', 0)))
        
        with open(CENTRAL_LABELS_FILE, 'w') as f:
            for label in all_labels:
                f.write(json.dumps(label) + '\n')
        
        print(f"\nMigration complete.")
        print(f"Processed {found_sessions} sessions.")
        print(f"Total records migrated: {total_records}")
        print(f"Central labels file created at: {CENTRAL_LABELS_FILE}")
    else:
        print("\nNo labels were found to migrate.")

if __name__ == "__main__":
    migrate_labels()
