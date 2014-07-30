tjob - Test Job Management Utility
==================================

* GitHub: https://github.com/ohmu/tjob
* Issues: https://github.com/ohmu/tjob/issues
* Pull requests: https://github.com/ohmu/tjob/pulls
* License: Apache 2.0, see ``LICENSE``

Building
========
#. Install Go >=1.2 and setup a proper ``$GOPATH``
#. ``make dep`` installs the dependency libraries
#. ``go build`` builds the stand-alone executable ``tjob``

Getting Started
===============
#. ``tjob runner add myjenkins --url https://jenkins.examples.com/jenkins --user myusername --insecure=true --ssh-key id_rsa``
#. ``tjob list --remote -j somejobname``
