#!/usr/bin/bash

/opt/singularity-2.6.0/bin/singularity run --app lammps /storage/lammps/lammps.avx512.simg 8 1 2>&1 | tee ~/lammps.log
