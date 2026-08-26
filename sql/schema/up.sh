#!/bin/bash

goose postgres "postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable" up
