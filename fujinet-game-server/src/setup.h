/**
 * Atari Game Server
 *
 * setup.h - Setup 
 *
 */

#ifndef SETUP_H
#define SETUP_H

#include <stdbool.h>

/**
 * @brief Set up server
 * @param argc CLI argument count 
 * @param argv CLI argument array
 * @return TRUE if setup completed, FALSE if not.
 */
bool setup(int argc, char* argv[]);

#endif /* SETUP_H */
