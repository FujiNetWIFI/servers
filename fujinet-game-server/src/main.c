/**
 * Atari Game Server
 *
 * main.c - Main program
 *
 */

#include <stdio.h>
#include "version.h"
#include "setup.h"
#include "run.h"
#include "done.h"
#include "context.h"

/**
 * Global server context
 */
Context context;

/**
 * Return code
 */
int ret;

void banner(int argc, char* argv[])
{
  printf("%s " VERSION " - Built " __DATE__" " __TIME__ "\n",argv[0]);
}

int main(int argc, char* argv[])
{
  banner(argc, argv);
  
  if (!setup(argc, argv))
    return ret;

  if (!run(argc, argv))
    return ret;

  if (!done(argc, argv))
    return ret;

  return 0;
}
