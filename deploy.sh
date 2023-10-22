  gcloud functions deploy rebug-pager \
    --gen2 \
    --runtime=go121 \
    --region=us-east1 \
    --source=. \
    --entry-point RebugPager \
    --trigger-http \
    --allow-unauthenticated