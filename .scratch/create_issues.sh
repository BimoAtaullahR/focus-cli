#!/bin/bash
ISSUE1_URL=$(gh issue create --title "Core SessionEngine & CLI Integration" --body-file .scratch/issue1.md --label "ready-for-agent")
echo "Created $ISSUE1_URL"
ISSUE1_NUM=$(basename "$ISSUE1_URL")

sed -i "s/\[INSERT_ISSUE_1_NUMBER\]/$ISSUE1_NUM/g" .scratch/issue2.md
ISSUE2_URL=$(gh issue create --title "SessionEngine Events & Task/Notification Updates di CLI" --body-file .scratch/issue2.md --label "ready-for-agent")
echo "Created $ISSUE2_URL"
ISSUE2_NUM=$(basename "$ISSUE2_URL")

sed -i "s/\[INSERT_ISSUE_2_NUMBER\]/$ISSUE2_NUM/g" .scratch/issue3.md
ISSUE3_URL=$(gh issue create --title "TUI Integration dengan SessionEngine" --body-file .scratch/issue3.md --label "ready-for-agent")
echo "Created $ISSUE3_URL"
