// Package repository contains all database access logic for the FITS processor.
// The old monolithic FITSRepository has been split into:
//   - FileRepository     (file_repository.go)
//   - HeaderRepository   (header_repository.go)
//   - MetadataRepository (metadata_repository.go)
//   - JobRepository      (job_repository.go)
//   - UserRepository     (user_repository.go)
//
// This file is kept for reference only. Do not add new methods here.
package repository
