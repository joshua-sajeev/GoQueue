# GoQueue Error Log

This file captures key concepts, patterns, and lessons learned while building GoQueue.
## 2025-12-05
```
FATAL:  role "root" does not exist
```
I got the above log from postgres container.  Below configuration fixed it.
```
test: [ "CMD", "pg_isready", "-q", "-d", "$POSTGRES_DB", "-U", "$POSTGRES_USER" ]
```

---
