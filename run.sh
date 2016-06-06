#!/bin/bash
set -e

APPENV=${APPENV:-gmunchenv}

/opt/bin/s3kms -r us-west-1 get -b opsee-keys -o dev/$APPENV > /$APPENV

source /$APPENV && \
  /opt/bin/s3kms -r us-west-1 get -b opsee-keys -o dev/$GMUNCH_CERT > /$GMUNCH_CERT && \
  /opt/bin/s3kms -r us-west-1 get -b opsee-keys -o dev/$GMUNCH_CERT_KEY > /$GMUNCH_CERT_KEY && \
  chmod 600 /$GMUNCH_CERT_KEY && \
  /gmunch
