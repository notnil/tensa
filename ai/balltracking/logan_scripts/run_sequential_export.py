import os
import subprocess
import argparse
import sys
import time

def parse_args():
    parser = argparse.ArgumentParser(description="Orchestrate sequential SVO export and post-sync.")
    parser.add_argument("input_dir", type=str, help="Directory containing camera SVO files (front.svo2, etc.)")
    parser.add_argument("output_dir", type=str, help="Root directory for exported data")
    parser.add_argument("--sample-interval", type=float, default=None, help="Sample interval in seconds")
    parser.add_argument("--fast-depth", action="store_true", help="Use PERFORMANCE depth mode")
    parser.add_argument("--keep-raw", action="store_true", help="Keep the raw camera folders after sync")
    parser.add_argument("--tolerance-ms", type=float, default=100.0, help="Sync tolerance in milliseconds")
    return parser.parse_args()

def main():
    args = parse_args()
    
    script_dir = os.path.dirname(os.path.abspath(__file__))
    cameras = ["front", "back", "left", "right"]
    
    # Check if SVO files exist
    svo_files = {}
    for cam in cameras:
        # Try both .svo and .svo2
        path2 = os.path.join(args.input_dir, f"{cam}.svo2")
        path1 = os.path.join(args.input_dir, f"{cam}.svo")
        
        if os.path.isfile(path2):
            svo_files[cam] = path2
        elif os.path.isfile(path1):
            svo_files[cam] = path1
        else:
            print(f"Error: Could not find SVO file for camera '{cam}' in {args.input_dir}")
            sys.exit(1)

    if not os.path.exists(args.output_dir):
        os.makedirs(args.output_dir, exist_ok=True)

    start_time = time.time()
    print(f"--- Starting Sequential Export of {len(cameras)} cameras ---")

    for cam in cameras:
        svo_path = svo_files[cam]
        print(f"\n>>> Exporting {cam.upper()} camera from {svo_path}...")
        
        cmd = [
            sys.executable, os.path.join(script_dir, "export_svo_single.py"),
            svo_path,
            args.output_dir,
            "--cam-name", cam
        ]
        if args.fast_depth:
            cmd.append("--fast-depth")
        if args.sample_interval:
            cmd.extend(["--sample-interval", str(args.sample_interval)])
            
        result = subprocess.run(cmd)
        if result.returncode != 0:
            print(f"Error: Export failed for camera {cam}")
            sys.exit(1)

    export_time = time.time() - start_time
    print(f"\n--- Export phase complete in {export_time:.2f} seconds ---")

    print("\n>>> Running Synchronization...")
    sync_cmd = [
        sys.executable, os.path.join(script_dir, "sync_exported_frames.py"),
        args.output_dir,
        "--tolerance-ms", str(args.tolerance_ms)
    ]
    if not args.keep_raw:
        sync_cmd.append("--move")
        
    result = subprocess.run(sync_cmd)
    if result.returncode != 0:
        print("Error: Synchronization failed")
        sys.exit(1)

    total_time = time.time() - start_time
    print(f"\n--- All operations complete in {total_time:.2f} seconds ---")
    print(f"Synced data available at: {os.path.join(args.output_dir, 'synced')}")

if __name__ == "__main__":
    main()

