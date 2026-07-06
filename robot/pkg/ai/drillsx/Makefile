# Google Cloud Transcoder helpers for portrait HLS (720p + 1080p)

# Configuration
PROJECT_ID ?= $(shell gcloud config get-value project 2>/dev/null)
LOCATION ?= us-central1

# Template (already in this directory)
TEMPLATE_ID ?= portrait-hls-720p-1080p
TEMPLATE_FILE ?= portrait-hls-720p-1080p.json

# Job-specific values (set these when running `make transcode`)
INPUT_URI ?=
OUTPUT_URI ?=

# Optional: output bucket for granting access (bucket name only, no gs:// prefix)
OUTPUT_BUCKET ?=

# Drills manifest upload
DRILLS_FILE ?= drills.json
DRILLS_GCS_URI ?= gs://example-public-bucket/tensa/drills/drills.json

# Upload and transcode
LOCAL_FILE ?=
SLUG_NAME ?=
ORIGINAL_ASSETS_BUCKET ?= gs://example-public-bucket/tensa/original-assets
DRILLS_VIDEOS_BUCKET ?= gs://example-public-bucket/tensa/drills/videos

.PHONY: help print-config enable-api create-template delete-template transcode jobs-list jobs-describe grant-output-bucket-access drills-upload upload-and-transcode

help:
	@echo "Targets:"
	@echo "  enable-api                   Enable Transcoder API in current project ($(PROJECT_ID))"
	@echo "  create-template              Create template $(TEMPLATE_ID) from $(TEMPLATE_FILE) in $(LOCATION)"
	@echo "  delete-template              Delete template $(TEMPLATE_ID) in $(LOCATION)"
	@echo "  transcode                    Create a job using template $(TEMPLATE_ID)"
	@echo "  jobs-list                    List recent jobs in $(LOCATION)"
	@echo "  jobs-describe JOB_ID=...     Describe a specific job"
	@echo "  grant-output-bucket-access   Grant Transcoder SA write access to OUTPUT_BUCKET"
	@echo "  upload-and-transcode          Upload local file to GCS and transcode (requires LOCAL_FILE and SLUG_NAME)"
	@echo "  drills-upload                 Upload local drills.json to GCS"
	@echo
	@echo "Variables:"
	@echo "  LOCATION?=$(LOCATION)"
	@echo "  TEMPLATE_ID?=$(TEMPLATE_ID)"
	@echo "  TEMPLATE_FILE?=$(TEMPLATE_FILE)"
	@echo "  INPUT_URI=gs://... (required for transcode)"
	@echo "  OUTPUT_URI=gs://... (required for transcode)"
	@echo "  OUTPUT_BUCKET=bucket-name (optional for access grant)"
	@echo "  DRILLS_FILE?=$(DRILLS_FILE) (local drills manifest)"
	@echo "  DRILLS_GCS_URI?=$(DRILLS_GCS_URI) (destination in GCS)"
	@echo "  LOCAL_FILE=path/to/file (required for upload-and-transcode)"
	@echo "  SLUG_NAME=slug-name (required for upload-and-transcode)"
	@echo
	@echo "Examples:"
	@echo "  make enable-api"
	@echo "  make create-template"
	@echo "  make transcode INPUT_URI=gs://example-public-bucket/tensa/original-assets/2025-10-15-Inside-Out-Forehand-v1.mov OUTPUT_URI=gs://example-public-bucket/tensa/drills/videos/2e316e7e-1b35-418c-b8cd-c10041f99689/"
	@echo "  make jobs-list"
	@echo "  make jobs-describe JOB_ID=1234567890"
	@echo "  make grant-output-bucket-access OUTPUT_BUCKET=tensa-media-public"
	@echo "  make drills-upload  # upload local drills.json to GCS"
	@echo "  make upload-and-transcode LOCAL_FILE=./video.mov SLUG_NAME=two-handed-backhand-grip"

print-config:
	@echo "PROJECT_ID=$(PROJECT_ID)"
	@echo "LOCATION=$(LOCATION)"
	@echo "TEMPLATE_ID=$(TEMPLATE_ID)"
	@echo "TEMPLATE_FILE=$(TEMPLATE_FILE)"
	@echo "INPUT_URI=$(INPUT_URI)"
	@echo "OUTPUT_URI=$(OUTPUT_URI)"
	@echo "OUTPUT_BUCKET=$(OUTPUT_BUCKET)"
	@echo "DRILLS_FILE=$(DRILLS_FILE)"
	@echo "DRILLS_GCS_URI=$(DRILLS_GCS_URI)"

enable-api:
	gcloud services enable transcoder.googleapis.com

create-template:
	gcloud transcoder templates create $(TEMPLATE_ID) \
	  --location=$(LOCATION) \
	  --file=$(TEMPLATE_FILE)

delete-template:
	gcloud transcoder templates delete $(TEMPLATE_ID) \
	  --location=$(LOCATION) \
	  --quiet

# Create an HLS job using the portrait template.
transcode: _require-input _require-output
	gcloud transcoder jobs create \
	  --location=$(LOCATION) \
	  --input-uri=$(INPUT_URI) \
	  --output-uri=$(OUTPUT_URI) \
	  --template-id=$(TEMPLATE_ID)

jobs-list:
	gcloud transcoder jobs list --location=$(LOCATION)

jobs-describe:
	@test -n "$(JOB_ID)" || (echo "ERROR: set JOB_ID=..." && exit 1)
	gcloud transcoder jobs describe $(JOB_ID) --location=$(LOCATION)

# Upload local drills.json to the public media bucket
drills-upload:
	@test -f "$(DRILLS_FILE)" || (echo "ERROR: $(DRILLS_FILE) not found" && exit 1)
	gsutil cp "$(DRILLS_FILE)" "$(DRILLS_GCS_URI)"

# Upload a local file to original-assets and transcode it to drills/videos/slug-name
upload-and-transcode: _require-local-file _require-slug-name
	@echo "Uploading $(LOCAL_FILE) to $(ORIGINAL_ASSETS_BUCKET)/..."
	@FILENAME=$$(basename "$(LOCAL_FILE)"); \
	GCS_INPUT_URI="$(ORIGINAL_ASSETS_BUCKET)/$$FILENAME"; \
	GCS_OUTPUT_URI="$(DRILLS_VIDEOS_BUCKET)/$(SLUG_NAME)/"; \
	echo "Input URI: $$GCS_INPUT_URI"; \
	echo "Output URI: $$GCS_OUTPUT_URI"; \
	gsutil cp "$(LOCAL_FILE)" "$$GCS_INPUT_URI" && \
	echo "Upload complete. Starting transcoding job..." && \
	gcloud transcoder jobs create \
	  --location=$(LOCATION) \
	  --input-uri="$$GCS_INPUT_URI" \
	  --output-uri="$$GCS_OUTPUT_URI" \
	  --template-id=$(TEMPLATE_ID) && \
	echo "Transcoding job created successfully!"

# Internal guards
_require-input:
	@test -n "$(INPUT_URI)" || (echo "ERROR: set INPUT_URI=gs://..." && exit 1)

_require-output:
	@test -n "$(OUTPUT_URI)" || (echo "ERROR: set OUTPUT_URI=gs://..." && exit 1)

_require-local-file:
	@test -n "$(LOCAL_FILE)" || (echo "ERROR: set LOCAL_FILE=path/to/file" && exit 1)
	@test -f "$(LOCAL_FILE)" || (echo "ERROR: $(LOCAL_FILE) not found" && exit 1)

_require-slug-name:
	@test -n "$(SLUG_NAME)" || (echo "ERROR: set SLUG_NAME=slug-name" && exit 1)


