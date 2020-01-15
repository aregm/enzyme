#!/usr/bin/bash

if grep `hostname` ~/clck/nodefile | grep -q head ; then
	echo running on frontend, executing mpi
	/opt/intel/compilers_and_libraries_2019.5.281/linux/mpi/intel64/bin/mpirun -f ~/Rhoc-nodefile "$0" "$@"
	exit $?
else
	echo running on worker
fi

source /opt/intel/compilers_and_libraries_2019.5.281/linux/bin/compilervars.sh intel64

cd ~/Rhoc-upload
/opt/intel/compilers_and_libraries_2019.5.281/linux/mkl/benchmarks/mp_linpack/xhpl_intel64_dynamic "$@"
