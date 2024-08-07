# The following assumes you have set up a project called "fujitzee" in Google Cloud

# It will create a cloud run deployment on the first run, or update the existing deployment on subsequent runs

# After you deploy, it will NOT contact the Lobby until you create an environment variable called GO_PROD=1

gcloud config set project fujitzee
gcloud run deploy fujitzee --source . --region=us-central1 --min-instances=1 --max-instances=1