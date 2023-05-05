#!/usr/bin/env python
"""
All global constants and variables used in the blackjack game.

Copyright (C) Torbjorn Hedqvist - All Rights Reserved
You may use, distribute and modify this code under the
terms of the MIT license. See LICENSE file in the project
root for full license information.

"""

# Standard imports
# import inspect  # To be used to print function name in log statements
import logging

# Set log level
logging.basicConfig(
    # filename='blackjack_debug.log',
    # filemode='w',  # overwrite previous logs
    # level=logging.DEBUG,
    # level=logging.INFO,
    # level=logging.WARNING,
    format="%(asctime)s:%(levelname)s:%(module)s:%(lineno)d:%(message)s"
    )

####################
# Global constants #
####################

# Paths
IMAGE_PATH = 'images/'
IMAGE_PATH_CARDS = 'images/cards/'
IMAGE_PATH_CHIPS = 'images/casino_chips/'
IMAGE_PATH_BUTTONS = 'images/buttons/'
SOUND_PATH = 'sounds/'
# previously using IMAGE_PATH = "./images/" which works as well

# The card back image used to print the dealers initial hidden card
CARDBACK_FILENAME = "cardback1.png"

# All button images

INACTIVE_BUTTON_FILENAME = "button_inactive.png"
ACTIVE_BUTTON_FILENAME   = "button_active.png"


# All chips images
CHIP_5_FILENAME_ON    = "chip_5_w85h85.png"
CHIP_5_FILENAME_OFF   = "chip_5_w85h85_fade.png"
CHIP_10_FILENAME_ON   = "chip_10_w85h85.png"
CHIP_10_FILENAME_OFF  = "chip_10_w85h85_fade.png"
CHIP_50_FILENAME_ON   = "chip_50_w85h85.png"
CHIP_50_FILENAME_OFF  = "chip_50_w85h85_fade.png"
CHIP_100_FILENAME_ON  = "chip_100_w85h85.png"
CHIP_100_FILENAME_OFF = "chip_100_w85h85_fade.png"

# Colors
GAME_BOARD_COLOR = (34, 139,  34)  # Nice TexasHoldem table color
GOLD_COLOR = (255, 215, 0)
BLACK_COLOR = (0, 0, 0)
WHITE_COLOR = (255, 255, 255)
BLUE_COLOR = (0, 0, 255)
GREEN_COLOR = (0, 255, 0)
RED_COLOR = (255, 0, 0)
YELLOW_COLOR = (255, 255, 0)
GREY_COLOR   = (192, 192, 192)

# Size, positions and gaps between objects on the game board
GAME_BOARD_SIZE = (800, 600)
GAME_BOARD_X_SIZE, GAME_BOARD_Y_SIZE = GAME_BOARD_SIZE

X_INDENT = 40
Y_INDENT = 20

FONT_HEIGHT = 25

GAP_BETWEEN_CARDS_HORIZ  = 14
GAP_BETWEEN_CARDS_VERT   = 0

GAP_BETWEEN_STACKS_HORIZ = 140
GAP_BETWEEN_STACKS_VERT  = (GAME_BOARD_Y_SIZE - Y_INDENT*2) / 5

GAP_BETWEEN_CHIPS = 10
GAP_BETWEEN_BUTTONS = 10
GAP_BETWEEN_SPLIT = 190

BOTTOM_CARD = GAME_BOARD_Y_SIZE - GAP_BETWEEN_STACKS_VERT - Y_INDENT

X1_CARD = 80
X2_CARD = X1_CARD - X_INDENT

X3_CARD = GAME_BOARD_X_SIZE - X_INDENT - GAP_BETWEEN_STACKS_HORIZ - 4*GAP_BETWEEN_CARDS_HORIZ
X4_CARD = X3_CARD + X_INDENT



PLAYER_CARD_START_POS = [ (int(GAME_BOARD_X_SIZE/2)-int(GAP_BETWEEN_STACKS_HORIZ/2),  BOTTOM_CARD,                             GAP_BETWEEN_CARDS_HORIZ, GAP_BETWEEN_CARDS_VERT),
                                               
                          (X1_CARD,                                                   BOTTOM_CARD - GAP_BETWEEN_STACKS_VERT  +FONT_HEIGHT, GAP_BETWEEN_CARDS_HORIZ, GAP_BETWEEN_CARDS_VERT),
                          (X2_CARD,                                                   BOTTOM_CARD - GAP_BETWEEN_STACKS_VERT*2,             GAP_BETWEEN_CARDS_HORIZ, GAP_BETWEEN_CARDS_VERT),
                          (X1_CARD,                                                   BOTTOM_CARD - GAP_BETWEEN_STACKS_VERT*3-FONT_HEIGHT, GAP_BETWEEN_CARDS_HORIZ, GAP_BETWEEN_CARDS_VERT),
                           
                          (int(GAME_BOARD_X_SIZE/2)-int(GAP_BETWEEN_STACKS_HORIZ/2),  BOTTOM_CARD - GAP_BETWEEN_STACKS_VERT*4,             GAP_BETWEEN_CARDS_HORIZ, GAP_BETWEEN_CARDS_VERT),
                           
                          (X3_CARD,                                                   BOTTOM_CARD - GAP_BETWEEN_STACKS_VERT*3-FONT_HEIGHT, GAP_BETWEEN_CARDS_HORIZ, GAP_BETWEEN_CARDS_VERT),
                          (X4_CARD,                                                   BOTTOM_CARD - GAP_BETWEEN_STACKS_VERT*2,             GAP_BETWEEN_CARDS_HORIZ, GAP_BETWEEN_CARDS_VERT),
                          (X3_CARD,                                                   BOTTOM_CARD - GAP_BETWEEN_STACKS_VERT  +FONT_HEIGHT, GAP_BETWEEN_CARDS_HORIZ, GAP_BETWEEN_CARDS_VERT)
                                                                                 
                           ]

PLAYER_NAME_START_POS = [
                          (PLAYER_CARD_START_POS[0][0],PLAYER_CARD_START_POS[0][1]),
                          (PLAYER_CARD_START_POS[1][0],PLAYER_CARD_START_POS[1][1]),
                          (PLAYER_CARD_START_POS[2][0],PLAYER_CARD_START_POS[2][1]),
                          (PLAYER_CARD_START_POS[3][0],PLAYER_CARD_START_POS[3][1]),
                          (PLAYER_CARD_START_POS[4][0],PLAYER_CARD_START_POS[4][1]),
                          (PLAYER_CARD_START_POS[5][0],PLAYER_CARD_START_POS[5][1]),
                          (PLAYER_CARD_START_POS[6][0],PLAYER_CARD_START_POS[6][1]),
                          (PLAYER_CARD_START_POS[7][0],PLAYER_CARD_START_POS[7][1])
                        ]
CHIPS_START_POS   = (700, 500)
BUTTONS_START_POS = (10, 500)
BUTTONS_START_POS_X, BUTTONS_START_POS_Y = BUTTONS_START_POS
STATUS_START_POS  = (480, 15)



# Timers in seconds
PAUSE_TIMER1 = 0.1
PAUSE_TIMER2 = 1
PAUSE_TIMER3 = 3

# Misc
NUM_OF_DECKS = 4
LOWEST_BET = 5
DEFAULT_PLAYER_BALANCE = 5000
COUNTING_HELP = False

MAX_PLAYERS = 8

