# The following assumes you have set up a project called "battleship" in Google Cloud

# It will create a cloud run deployment on the first run, or update the existing deployment on subsequent runs

# After you deploy, it will NOT contact the Lobby until you create an environment variable called GO_PROD=1

gcloud config set project battleship
gcloud run deploy battleship --source . --region=us-central1 --min-instances=0 --max-instances=1 --revision-suffix=""