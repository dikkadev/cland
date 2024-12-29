#!/bin/bash

FILEPATH=./tmp/input/test.txt

cat > $FILEPATH <<- EOF
Cool Things
testing: true
---
a really awsome message
EOF

sleep 1
rm $FILEPATH
