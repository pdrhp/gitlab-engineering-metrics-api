# Fair Attribution Metrics Guide

## Overview

As of 2026-04-11, the user performance API uses **fair attribution** for individual metrics via `vw_assignee_cycle_time`.

### The Problem We Solved

Previously, when multiple people worked on the same issue at different times, the LAST assignee received credit/blame for ALL the time. This created "poisoned data" for performance evaluation.

### The Solution

Now, each assignee receives credit ONLY for the time they actually had the issue assigned.

## What Changed

### Before (Unfair Attribution)

Issue #1766 with 6 assignees, 1261 total hours:
- torezan: 1261h (credited with ALL time) ❌
- danilo: 1261h (credited with ALL time) ❌
- vitorfsampaio: 1261h (credited with ALL time) ❌

### After (Fair Attribution)

Issue #1766 with 6 assignees, 1261 total hours:
- torezan: 276h active (their actual time) ✅
- danilo: 0h active (only 90h backlog) ✅
- vitorfsampaio: 0h active (173h total, mostly blocked) ✅

## Response Changes

### New Field: `individual_performance`

```json
{
  "user": { "username": "nevez" },
  "individual_performance": {
    "username": "nevez",
    "issues_assigned": 35,
    "issues_contributed": 33,
    "total_active_cycle_hours": 27626.18,
    "active_work_pct": 99.22,
    "total_dev_hours": 20145.32,
    "total_qa_hours": 7480.86,
    "total_blocked_hours": 145.00,
    "total_backlog_hours": 250.00,
    "total_hours_as_assignee": 28021.18,
    "p50_active_cycle_hours": 245.50,
    "p95_active_cycle_hours": 892.30
  }
}
```

### Full Schema

```typescript
interface IndividualPerformance {
  username: string;
  issues_assigned: number;           // Total issues where user was assignee
  issues_contributed: number;        // Issues with active work (IN_PROGRESS + QA_REVIEW)
  total_active_cycle_hours: number;  // Hours in IN_PROGRESS + QA_REVIEW
  active_work_pct: number;           // Percentage of time in active states (0-100)
  total_dev_hours: number;           // Hours in IN_PROGRESS state
  total_qa_hours: number;            // Hours in QA_REVIEW state
  total_blocked_hours: number;       // Hours in BLOCKED state
  total_backlog_hours: number;       // Hours in BACKLOG state
  total_hours_as_assignee: number;   // Sum of all time as assignee
  p50_active_cycle_hours: number;    // Median cycle time per issue
  p95_active_cycle_hours: number;    // 95th percentile cycle time
}
```

### Key Metrics Explained

| Field | Description | Why It Matters |
|-------|-------------|----------------|
| `issues_assigned` | Total issues where user was assignee | Shows workload |
| `issues_contributed` | Issues with active work (IN_PROGRESS + QA_REVIEW) | Shows actual contribution |
| `active_work_pct` | % of time in active states | Efficiency indicator |
| `total_active_cycle_hours` | Hours in IN_PROGRESS + QA_REVIEW | Real work done |
| `total_dev_hours` | Hours spent in IN_PROGRESS state | Development effort |
| `total_qa_hours` | Hours spent in QA_REVIEW state | Quality assurance effort |
| `total_blocked_hours` | Hours in BLOCKED state | Time lost to blockers |
| `total_backlog_hours` | Hours in BACKLOG state | Time waiting to start |
| `total_hours_as_assignee` | Sum of all time as assignee | Total ownership time |
| `p50_active_cycle_hours` | Median active cycle time per issue | Typical issue size |
| `p95_active_cycle_hours` | 95th percentile active cycle time | Outlier detection |

### Fair vs Unfair Comparison

| Scenario | Old Approach | Fair Attribution |
|----------|--------------|------------------|
| Issue with 1 assignee | ✅ Correct | ✅ Correct |
| Issue with 3 assignees | ❌ 100% credit to 1 person | ✅ Credit divided fairly |
| Assignee entered at end | ❌ Looks like they did everything | ✅ Gets only their hours |
| Frequent handoffs | ❌ Metrics distorted | ✅ Each person gets their time |
| Assignee with 20% active work | ❌ Not detectable | ✅ Visible in `active_work_pct` |

## When to Use Fair Metrics

### ✅ Use `individual_performance` for:

- **Performance reviews** - Accurate picture of individual contribution
- **Identifying bottlenecks** - Find who has high blocked time
- **Capacity planning** - Understand actual workload per person
- **Recognizing individual contributions** - Credit goes to the right person
- **Finding assignees who never actively worked on issues** - Low `active_work_pct`
- **Detecting ghost assignments** - `issues_assigned > 0` but `issues_contributed = 0`
- **Understanding work distribution** - Dev vs QA time breakdown

### ❌ Do NOT use for:

- **Project velocity** - Use project-level endpoints
- **Team throughput** - Aggregate issues, not cycle time
- **Lead time calculations** - Use `vw_issue_lifecycle_metrics`
- **Release planning** - Use aggregate metrics, not individual

## Migration Notes

### For API Consumers

**No breaking changes** - existing fields remain unchanged. The `individual_performance` object is **additional** data.

```bash
# Old response still works
GET /api/v1/users/{username}/performance

# New field added
{
  "user": { ... },
  "individual_performance": { ... }  // ← New, optional field
}
```

### For Project-Level Metrics

**No changes** - endpoints like `GET /api/v1/projects/{id}/metrics` continue to use `vw_issue_lifecycle_metrics` for aggregate throughput/velocity.

### Backward Compatibility

| Aspect | Status |
|--------|--------|
| Existing fields | ✅ Unchanged |
| Response format | ✅ Compatible |
| Query parameters | ✅ No changes |
| Authentication | ✅ No changes |
| Rate limits | ✅ No changes |

## Example Queries

### Find Assignees with Low Active Work %

```bash
GET /api/v1/users/{username}/performance
# Check individual_performance.active_work_pct < 50
# Indicates: issue was assigned but not touched, or assignee was formal only
```

### Identify High Contributors

```bash
# Sort by individual_performance.issues_contributed
# Look for active_work_pct > 80%
```

### Detect "Ghost Assignees"

```bash
# Users with issues_assigned > 0 but issues_contributed = 0
# These users were assigned but never did active work
```

### Compare Dev vs QA Effort

```bash
# Use individual_performance.total_dev_hours and total_qa_hours
# Ratio indicates work type distribution
```

### Find Blocked Work

```bash
# Filter by individual_performance.total_blocked_hours > threshold
# Identify systemic blocker issues
```

## FAQ

**Q: Why is `issues_contributed` < `issues_assigned`?**  
A: User was formally assigned but didn't do active work (IN_PROGRESS/QA_REVIEW). Issue may have been blocked, or another person did the actual work.

**Q: What if `active_work_pct` is 0%?**  
A: User was assignee but issue stayed in BACKLOG/BLOCKED during their assignment. They never actively worked on it.

**Q: Can I sum `total_active_cycle_hours` across team members?**  
A: Yes! Unlike old approach, fair metrics are additive without double-counting.

**Q: Why are my "top performer" numbers different from before?**  
A: Our system assigns credit ONLY for each assignee's real time. If another system assigns all cycle time to one person, numbers will differ (and be less fair).

**Q: How do I handle assignees with `active_work_pct < 50%`?**  
A: Investigate case-by-case. Could mean:
- Issue was assigned but never touched
- Assignee was formal, but someone else executed
- External blockers prevented work
- Issue was reassigned before work started

**Q: Why don't project metrics use fair attribution?**  
A: Project-level metrics measure throughput (issues completed), not individual contribution. Fair attribution matters for personal performance, not aggregate velocity.

**Q: What happens if an issue has only one assignee?**  
A: Fair attribution equals traditional attribution. The single assignee gets 100% of the time, which is correct.

**Q: Are percentiles calculated per issue or across all issues?**  
A: Percentiles (`p50_active_cycle_hours`, `p95_active_cycle_hours`) are calculated across all issues the user contributed to. They show the distribution of issue sizes for that person.

## Technical Details

### Database Views

This implementation uses:

| View | Purpose |
|------|---------|
| `vw_assignee_cycle_time` | Per-assignee, per-issue cycle time breakdown |
| `vw_individual_performance_metrics` | Aggregated metrics per assignee |

### Schema Relationships

```
gitlab_issues
    └── gitlab_issue_assignees (N:M relationship)
            └── vw_assignee_cycle_time (calculates time per assignee)
                    └── vw_individual_performance_metrics (aggregates per user)
```

### SQL Example

```sql
SELECT 
    assignee_username,
    issues_assigned,
    issues_contributed,
    total_active_cycle_hours,
    active_work_pct
FROM vw_individual_performance_metrics
WHERE assignee_username = 'nevez';
```

### View Definitions

**vw_assignee_cycle_time** - Calculates time each assignee held an issue:

```sql
SELECT 
    i.id as issue_id,
    a.assignee_username,
    -- Time calculation logic based on state transitions
    -- while this assignee was responsible
FROM gitlab_issues i
JOIN gitlab_issue_assignees a ON i.id = a.issue_id
```

**vw_individual_performance_metrics** - Aggregates per-user statistics:

```sql
SELECT 
    assignee_username,
    COUNT(DISTINCT issue_id) as issues_assigned,
    SUM(active_cycle_hours) as total_active_cycle_hours,
    -- Additional aggregations
FROM vw_assignee_cycle_time
GROUP BY assignee_username
```

### Migration

- **Migration:** `000017_assignee_cycle_time`
- **Date:** 2026-04-11
- **Breaking Changes:** None (additive only)
- **Rollback:** Safe - only adds new fields, doesn't modify existing behavior

### Performance Considerations

| Aspect | Impact |
|--------|--------|
| Query complexity | Low - views are pre-aggregated |
| Response size | +2-3KB per user response |
| Database load | Minimal - uses indexed views |
| Cache behavior | No change - cache keys unchanged |

### API Endpoints Affected

| Endpoint | Change |
|----------|--------|
| `GET /api/v1/users/{username}/performance` | ✅ Adds `individual_performance` field |
| `GET /api/v1/users/{username}/performance/cycle-time` | ✅ Uses fair attribution |
| `GET /api/v1/projects/{id}/metrics` | ❌ No change (aggregate metrics) |
| `GET /api/v1/projects/{id}/throughput` | ❌ No change (issue counts) |

---

**Questions?** Contact: Time de Plataforma - Canal #data-engineering

**Last Updated:** 2026-04-11  
**Version:** 1.0.0  
**Related Migrations:** 000017_assignee_cycle_time
