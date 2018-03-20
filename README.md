# SubaruWebQL
An experimental port of SubaruWebQL from C/C++ to Go

The purpose of this project is to discover pitfalls and limits of Go when used in interactive client-server applications at the Japanese Virtual Observatory (JVO). This project is not meant for general public use. It can only be run internally as part of the JVO Portal since it connects to various other internal network services.

During porting of SubaruWebQL to Go one disadvantage of GO has been found: the lack of pointer arithmetic. Whilst this makes Go "safe", it takes away a lot of power from a developer more used to assembler, C as well as VHDL FPGA hardware design.

It seems Google has created a language that is easy to learn, fast to develop in but lacks low-level features that power users are accustomed to in C/C++ or Rust.

The other gripe with Go is the lack of OpenMP support.

In the future the author might stop any further porting from C/C++ to Go and either stick with C/C++ or explore Rust. Rust seems to be closer to C/C++ than Go, and Rust seems to permit low-level memory manipulations.

*AN UPDATE*

Recently the author has been trying out Rust to see what is all the fuss with it. See the project below. The first impression of Rust is that it seems to be too strict. A *careful* programmer can write "safe" and crash-proof C/C++ code without losing time having to battle the Rust compiler.

https://github.com/jvo203/subaru_web_ql
