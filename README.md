# File Hosting and CDN Infrastructure with AWS

Coursework exploring static file hosting, content delivery networks, access control, caching, and cloud infrastructure fundamentals using Amazon S3, CloudFront, and IAM.

> **Note:** This repository is a fork of the project repository used in Boot.dev's *Learn File Servers and CDNs with S3 and CloudFront* course. The work here was completed as guided coursework.

## Overview

This project focused on the architecture behind serving static files reliably at scale:

```text
Client → CloudFront CDN → Amazon S3 Origin
```

Amazon S3 provides durable object storage for static assets, while CloudFront distributes cached copies through edge locations closer to users. IAM policies, users, groups, and roles control access to AWS resources.

## Learning Outcomes

- **Object Storage with Amazon S3**
  - Created and managed S3 buckets and stored static objects.
  - Worked with object keys, bucket-level configuration, and file accessibility.
  - Explored the distinction between storage origins and public delivery endpoints.

- **Identity and Access Management**
  - Created IAM users, groups, roles, access keys, and custom policies.
  - Applied least-privilege access principles to AWS resources.
  - Used identity-based policies to define permissions for IAM users and roles, and resource-based bucket policies to control access to S3 resources.

- **Content Delivery with CloudFront**
  - Created a CloudFront distribution backed by an S3 origin.
  - Served static files through a CDN rather than directly from object storage.
  - Explored edge caching, cache invalidation, and origin requests.

- **Web Delivery and Caching**
  - Examined how HTTP caching affects static asset delivery.
  - Learned how CDNs reduce latency and decrease load on origin infrastructure.
  - Investigated cache-control behavior and the tradeoff between freshness and performance.

- **Cloud Infrastructure Lifecycle**
  - Practiced provisioning and removing cloud resources.
  - Considered the cost and security implications of unused AWS infrastructure.
  - Learned the dependency order involved in disabling and deleting cloud resources.

## Implementation Highlights

- Generated S3 object keys and CloudFront URLs for uploaded video assets.
- Created temporary local files before uploading objects with the Go AWS SDK.
- Persisted uploaded asset metadata and delivery URLs in the application database.
- Separated object storage concerns from application data, storing references rather than file contents in PostgreSQL.
- Used CloudFront as the public delivery layer while S3 remained the origin.

## Technologies

| Technology | Purpose |
|---|---|
| Go | Course language and tooling |
| Amazon S3 | Static object storage and origin hosting |
| Amazon CloudFront | CDN and edge caching |
| AWS IAM | Access control through users, groups, roles, and policies |
| AWS CLI | AWS resource management and inspection |
| Git | Version control |

## Key Concepts

### Origin Storage vs. CDN Delivery

S3 stores the source files. CloudFront acts as a caching layer between the client and S3, serving content from geographically distributed edge locations when possible.

```text
First request:
Client → CloudFront → S3

Later cached requests:
Client → CloudFront edge cache
```

### Least-Privilege Access

IAM policies should grant only the actions and resources required for a task. This reduces the potential impact of leaked credentials or incorrectly configured applications.

### Cache Invalidation

CDNs improve performance by retaining copies of files. When a file changes, cached versions may need to expire naturally, use versioned filenames, or be explicitly invalidated.

## Repository Context

This is not intended to be a standalone production application. It documents hands-on AWS infrastructure work completed through Boot.dev coursework, including the AWS configuration and resource-management concepts practiced throughout the course.

## Acknowledgments

Built as part of Boot.dev's [Back-End Development Path](https://www.boot.dev/tracks/backend), specifically the [Learn File Servers and CDNs with S3 and CloudFront](https://www.boot.dev/courses/learn-file-servers-s3-cloudfront-golang) course.

The original repository and course material belong to Boot.dev. This fork contains my completed coursework and documentation.