# The following assumes you have set up a project called "fujitzee" in Google Cloud
gcloud config set project fujitzee
gcloud run deploy fujitzee --source . --region=us-central1 --min-instances=1 --max-instances=1