terraform {
  backend "s3" {
    bucket = "tf-backend-20210415"
    key    = "tidus-api/uat"
    region = "ap-southeast-1"
  }
}