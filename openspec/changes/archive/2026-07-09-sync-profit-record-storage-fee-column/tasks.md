## 1. Storage Spec Synchronization

- [x] 1.1 Update the `profit-record-storage` delta spec to describe the current eight-column storage header with the `fee` field
- [x] 1.2 Document persisted `fee` semantics for default multiplier usage versus explicit multiplier usage
- [x] 1.3 Document compatibility behavior for legacy five-column files and intermediate seven-column files without `fee`

## 2. Verification

- [x] 2.1 Cross-check the updated spec wording against `internal/domain/profit.go` and `internal/storage/storage.go`
- [x] 2.2 Validate the new OpenSpec change so it is ready for later apply and sync work
