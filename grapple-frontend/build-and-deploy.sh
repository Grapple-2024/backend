#!/bin/bash
set -e

echo "WARNING: this will build and deploy the current code in this repository. Do you want to proceed? (Y/n)"
read choice; if [ $choice != "Y" ]; then echo aborting; exit 1; fi

echo "exec: npm run install"
npm install

echo "exec: npm run build"
npm run build

echo "exec: pm2 restart 'npm run start'"
pm2 restart "npm run start"
