' fetch-latest-hidden.vbs - run fetch-latest.ps1 on a schedule without any console window.
' Windows 10/11 (Windows Script Host is part of the OS). scripts\register-fetch-task.ps1 registers
' this file as the scheduled task's action, started by wscript.exe: wscript.exe owns no console, so
' powershell.exe and every ssh.exe/scp.exe it starts inherit none and Windows never creates one - no
' black window, not even for a split second.
'
'   wscript.exe //nologo //B fetch-latest-hidden.vbs "<path to fetch-latest.ps1>" [extra args...]
'
' The first argument is the PowerShell script; every further argument is passed to it unchanged.
' The run window is hidden and waited for, so the task's duration and last-run result are the fetch's.
Option Explicit
Dim sh, args, i, commandLine
Set sh = CreateObject("WScript.Shell")
If WScript.Arguments.Count < 1 Then WScript.Quit 2
commandLine = "powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File " & Quote(WScript.Arguments(0))
For i = 1 To WScript.Arguments.Count - 1
    commandLine = commandLine & " " & Quote(WScript.Arguments(i))
Next
' 0 = hidden window, True = wait for it and return its exit code
WScript.Quit sh.Run(commandLine, 0, True)

Function Quote(text)
    Quote = """" & Replace(text, """", """""") & """"
End Function
