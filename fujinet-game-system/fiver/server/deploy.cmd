call gcloud config set project five-card-stud-383623
gcloud run deploy mock-server --source . --region=us-central1 --min-instances=1 --max-instances=1