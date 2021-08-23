# Node Module BOM Generator Buildpack

The Node Module BOM Generator CNB generates a bill of materials for all node
modules in an app image. The buildpack installs the [CycloneDX Node Module
tool](https://github.com/CycloneDX/cyclonedx-node-module) into a  layer for
usage within the buildpack, and is cached for subsequent builds. The tool is then
used to generate a Bill of Materials for all node modules found within the
application source directory.

## Integration

The Node Module BOM Generator CNB provides nothing. It detects on the presence
of a `node_modules` directory in the application source directory and requires
`node`.

If there is no `node_modules` directory in the application source directory,
`node_modules` (provided by the [NPM Install
CNB](https://github.com/paketo-buildpacks/npm-install) or [Yarn Install
CNB](https://github.com/paketo-buildpacks/yarn-install)) are also required for
detection.

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh --version <version-number>
```

This will create a `buildpackage.cnb` file under the `build` directory which you
can use to build your app as follows:
`pack build <app-name> -p <path-to-app> -b build/buildpackage.cnb`

## Configurations

### Opting Out

Users can opt out of this buildpack by setting the `BP_ENABLE_MODULE_BOM`
environment variable during container build-time to `false` either directly
(ex. `pack build my-app --env BP_ENABLE_MODULE_BOM=false`) or through a
[`project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md).

```shell
$BP_ENABLE_MODULE_BOM=false
```

The default value of `BP_ENABLE_MODULE_BOM` is `true`.

## Run Tests

To run all unit tests, run:
```
./scripts/unit.sh
```

To run all integration tests, run:
```
/scripts/integration.sh
```
