# Management api

Test with curl:
```bash
curl -v -H"X-Amz-Content-Sha256: UNSIGNED-PAYLOAD"\
  --request GET \
  --user "$AWS_ACCESS_KEY:$AWS_SECRET_ACCESS_KEY" \
  --aws-sigv4 "aws:amz:us-east-1:s3" \
  --header "Content-Type: application/json" \
  "127.0.0.1:8082/users"
```
