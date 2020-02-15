nodejs-packaging

------

Server-side tool to really package the nodejs modules.

## Development

Test nodejs-packaging: rename /home/abuild in main.go to yours and run rpmbuild -ba xxx.spec

Test nodejs-require: rpm -ql gulp | /usr/lib/rpm/rpmdeps -R --rpmfcdebug -vv 2>&1 | grep "^D: Executing" | head -n 5
