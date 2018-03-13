#!/bin/sh

set -e

# set AWS S3 details through ENV 
bucket=${S3_BUCKET}
s3Key=${AWS_ACCESS_KEY_ID}
s3Secret=${AWS_SECRET_ACCESS_KEY}

# set environment through ENV
environment=${RUNTIME_ENVIRONMENT}

# set service backup URL through ENV
serviceName=${SERVICE_NAME}
serviceBackupURL=${SERVICE_BACKUP_URL}

# download backup from service
curl -sL "${serviceBackupURL}" > backup.db
mkdir -p ${environment}/${serviceName}
file=${environment}/${serviceName}/backup.tar.gz
tar -czf "${file}" backup.db 

# prepare AWS S3 request
resource="/${bucket}/${file}"
contentType="application/x-compressed-tar"
dateValue=$(date -R)
stringToSign="PUT\n\n${contentType}\n${dateValue}\n${resource}"
signature=$(echo -en ${stringToSign} | openssl sha1 -hmac ${s3Secret} -binary | base64)

# backup file
curl -X PUT -T "${file}" \
  -H "Host: ${bucket}.s3.amazonaws.com" \
  -H "Date: ${dateValue}" \
  -H "Content-Type: ${contentType}" \
  -H "Authorization: AWS ${s3Key}:${signature}" \
  https://${bucket}.s3.amazonaws.com/${file}

echo "Backup successfully completed!"