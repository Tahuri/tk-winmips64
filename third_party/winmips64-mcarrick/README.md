# WinMIPS64

WinMIPS64 is an instruction set simulator, and was designed as a replacement for the popular Microsoft Windows utility WinDLX. 
The classic text *Computer Architecture - a Quantitative Approach, by Hennessy & Patterson*, 5th edition from its 3rd edition switched from the 32-bit DLX architecture, to the 64-bit MIPS architecture. 
Hence the need for a new teaching tool. WinDLX had a very nice friendly user interface, including a full graphical simulation of the 5-stage pipeline. Therefore it was decided to create a similar 
tool for the MIPS64 with a very similar interface.

If you know how to use WinDLX then the look-and-feel of WinMIPS64 will be very familiar. However there are a few changes.

Forwarding is indicated by colouring the forwarded register. The colour indicates the stage in the pipeline that it is being forwarded from.

A delay slot can be implemented if desired. A simple branch-target-buffer can also be simulated. A << in the code window beside a jump or branch instruction indicates that it is predicted as being taken.

The full instruction set is not currently implemented. In particular there is no support for single precision floating-point, or "paired single" floating-point.

Registers and memory can be changed by left-double-clicking on the target. Right-double-clicking on memory displays it in double-precision decimal floating point format.
Break-points are set and cleared by simply left-double-clicking on the code window.
 
The software is available as a Visual Studio C++ project, a simple stand-alone DOS assembler, and several example programs. It may be used freely without restriction. If you find any bugs, or 
would like to see an additional feature, please let me know at mscott@indigo.ie

PDF Documentation is available in the */doc* folder, along with a couple of lab exercises *ex2.pdf* and *ex3.pdf*

# Install

Copy the repository contents to a working directory, and build the project *WinMIPS64.sln* from source using Visual Studio 2022 C++ for x64. This will create the windows executable.


Mike Scott, March 2025

My Father 
	http://indigo.ie/~mscott/jhs.html
