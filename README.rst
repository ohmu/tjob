tjob - Test Job Management Utility
==================================

* GitHub: https://github.com/ohmu/tjob
* Issues: https://github.com/ohmu/tjob/issues
* Pull requests: https://github.com/ohmu/tjob/pulls
* License: Apache 2.0, see ``LICENSE``

Building
========
#. Install Go >=1.2 and setup a proper ``$GOPATH`` (e.g. ``export GOPATH=~/gopath``)
#. ``go get github.com/ohmu/tjob`` pulls the sources (and dependencies) under ``$GOPATH/src/``
#. ``go install github.com/ohmu/tjob`` builds the stand-alone executable to ``$GOPATH/bin/tjob``

Getting Started
===============
#. ``tjob runner add myjenkins --url https://jenkins.examples.com/jenkins --user myusername --insecure=true --ssh-key id_rsa``
#. ``tjob list --remote -j somejobname``
