There should be a generic configurable OSC target that can send OSC anywhere. There should also be a ready configured OSC for grandma3 that has easy helpers for the grandma3 commands like flash, temp, go, on off etc which the users use instead of writing raw commands to send via OSC.
When configuring the sources, the users creates an instance of that source for example grandma3 console XY and then configures the IP, port etc for the OSC protocol and then can reuse that source intance for mapping and doesnt have to configure IP etc when mapping every single button on the board.
There should be no top level grandma3 code or documentation since its solely one of the modules that can be used within the software. All code and documentation of that module should be within the module. The software at the top level should be completely generic.
The software should have all the necessary apis for the modules but should be complete without any modules. The software should be able to run without any modules and the user can add modules as needed. The software should be able to run with grandma3 module only, or with midi module only or with any other module only. The software should be able to run with multiple modules at the same time.

--

The types for communication from backend/frontend should be generated from the backend and not statically typed in the frontend.

--

The software should check for updates and notify the user if a new version is available. The user should be able to download and install the update from within the software. The software should also have an option to check for updates automatically on startup.
