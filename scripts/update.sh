#!/bin/bash

set -e

git fetch && git reset --hard origin/main && docker compose up -d --build