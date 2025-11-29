# The following assumes you have created a new empty project called "fuij-battleship" in Google Cloud
#
# It will create a cloud run deployment on the first run, or update the existing deployment on subsequent runs
#
# After you deploy, it will NOT contact the Lobby unless you create an environment variable called GO_PROD=1

# Required - set the current project. I use a unique project per server for easy billing tracking
gcloud config set project fuji-battleship

# Initial deployment of service - make sure everything is working
#gcloud run deploy fuji-battleship --source . --region=us-central1 --min-instances=0 --max-instances=1 --revision-suffix="" --cpu-boost --execution-environment=gen1 --memory=128Mi

# Production deployment - contacts the Lobby . Use this going forward once everthing is tested
gcloud run deploy fuji-battleship --set-env-vars GO_PROD=1 --source . --region=us-central1 --min-instances=0 --max-instances=1 --revision-suffix="" --cpu-boost --execution-environment=gen1 --memory=128Mi