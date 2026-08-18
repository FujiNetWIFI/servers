# Basic instructions to setup a project in google cloud run
# 1 (Optional) Create a dns entry that CNAMEs to ghs.googlehosted.com. Example: server.example.com
# 2 In google cloud console, create a new empty project. Take note of the project ID, which must be used below
# 3 Run the initial deployment of the service below. This will create a cloud run deployment on the first run, or update the existing deployment on subsequent runs
# 4 (Optional) If you created a dns entry in step 1, you can add a custom domain to the "cloud run" service. This is done in the cloud run console, and will require you to verify ownership of the domain. Once verified, you can add the custom domain to the service.

# Required - set the current project. Google limits projects so create a common fujinet-servers project for long term use.
gcloud config set project fujitzee

# Initial deployment of service - make sure everything is working
#gcloud run deploy fujirkle --source . --region=us-central1 --min-instances=0 --max-instances=1 --revision-suffix="" --cpu-boost --execution-environment=gen1 --memory=128Mi

# Production deployment - contacts the Lobby . Use this going forward once everthing is tested
gcloud run deploy fujirkle --set-env-vars GO_PROD=1 --source . --region=us-central1 --min-instances=0 --max-instances=1 --revision-suffix="" --cpu-boost --execution-environment=gen1 --memory=128Mi
