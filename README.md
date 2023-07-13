# enzyme
X-1 ML System anywhwere

## Install

OS - Rocky Linux 9.
Docker should be installed. 

```
git clone -b x1 https://github.com/aregm/enzyme.git
cd enzyme
./scripts/deploy/kind.sh
```

## Quick start

The cluster's endpoints are accessible only from localhost:

* http://dashboard.localtest.me
* http://jupyter.localtest.me
* http://minio.localtest.me
* http://prefect.localtest.me

In your browser, navigate to http://jupyter.localtest.me.

### Define a flow

Currently, ICL uses [Prefect](https://docs.prefect.io/) for defining basic workflow building blocks: [flow and tasks](https://docs.prefect.io/tutorials/first-steps/#flows-tasks-and-subflows).

Create a Python file `my_flow.py` that defines a single flow `my_flow`: 

```python
from prefect import flow

@flow
def my_flow():
    print('Hello from my_flow')
```

Note this is a regular Python file, so it can be developed, tested, and executed locally.

### Deploy and run a flow

The following code deploys and runs flow `my_flow` in the default infrastructure:

```python
import x1

program = await x1.deploy('my_flow.py')
await program.run()
```

## Links

* [docs/kind.md](https://github.com/aregm/enzyme/blob/x1/docs/kind.md)
