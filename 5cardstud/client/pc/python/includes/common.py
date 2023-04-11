#!/usr/bin/env python
"""
All common support functions and classes used in the black jack game

Copyright (C) Torbjorn Hedqvist - All Rights Reserved
You may use, distribute and modify this code under the
terms of the MIT license. See LICENSE file in the project
root for full license information.

"""

# Standard imports
import sys
import os
import pygame
import inspect  # To be used to print function name in log statements

# Local imports
MAIN_DIR = (os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
sys.path.insert(1, os.path.join(MAIN_DIR, 'includes'))
from globals import *


##########################
# Common support classes #
##########################


class ImageDB:
    """
    Instantiating this class into an object will create a singleton object
    which contains a library (dict) that stores the images when loaded.
    This will avoid reloading the image every time the function is called
    in the main game loop.
    Usage:
    instance = ImageDB.get_instance()
    image = instance.get_image(path)
    Or:
    image = ImageDB.get_instance().get_image(path)

    """
    instance = None

    @classmethod
    def get_instance(cls):
        """
        If instance is None create an instance of this class
        and return it, else return the existing instance.

        :return: An ImageDB instance.

        """
        if cls.instance is None:
            cls.instance = cls()
        return cls.instance

    def __init__(self):
        self.image_library = {}

    def get_image(self, path):
        """
        If the image exists in the dictionary it will be returned.
        If image is not found in the dictionary it will be loaded from
        the file system or throw exception if not found.

        :param path: <string> containing the absolute directory path \
        to where the expected image is located.
        :return: An image in pygame Surface object format.

        """
        logging.debug(inspect.stack()[0][3] + ':' + 'enter')

        image = self.image_library.get(path)
        if image is None:
            logging.info(inspect.stack()[0][3] + ':' + path)
            canonicalized_path = path.replace('/', os.sep).replace('\\', os.sep)
            image = pygame.image.load(canonicalized_path)
            self.image_library[path] = image
        return image


class SoundDB:
    """
    Instantiating this class into an object will create a singleton object
    which contains a library (dict) that stores the sounds when loaded.
    This will avoid reloading the sound every time the function is called
    in the main game loop.
    Usage:
    instance = SoundDB.get_instance()
    sound = instance.get_sound(path)
    Or:
    sound = SoundDB.get_instance().get_sound(path)

    """
    instance = None

    @classmethod
    def get_instance(cls):
        """
        If instance is None create an instance of this class
        and return it, else return the existing instance.

        :return: An SoundDB instance.

        """
        if cls.instance is None:
            cls.instance = cls()
        return cls.instance

    def __init__(self):
        logging.info(inspect.stack()[0][3] + ':' + 'SoundDb instance created')
        self.sound_library = {}

    def get_sound(self, path):
        """
        If the sound exists in the dictionary it will be returned.
        If sound is not found in the dictionary it will be loaded from
        the file system or throw exception if not found.

        :param path: <string> containing the absolute directory path \
        to where the expected sound is located.
        :return: An sound in pygame Surface object format.

        """
        logging.debug(inspect.stack()[0][3] + ':' + 'enter')

        sound = self.sound_library.get(path)
        if sound is None:
            logging.info(inspect.stack()[0][3] + ':' + path)
            canonicalized_path = path.replace('/', os.sep).replace('\\', os.sep)
            sound = pygame.mixer.Sound(canonicalized_path)
            self.sound_library[path] = sound
        return sound



class CommonVariables:
    """
    Instantiating this class into an object will create a singleton object
    containing all common variables to be passed around by reference
    between the main game loop and the various fsm states.

    """
    instance = None

    @classmethod
    def get_instance(cls):
        """
        If instance is None create an instance of this class
        and return it, else return the existing instance.

        :return: A CommonVariables instance.

        """
        if cls.instance is None:
            cls.instance = cls()
        return cls.instance

    def __init__(self):
        """
        Instantiate a singleton object with all attributes set to None.
        To be populated by the caller.

        """
        self.done 					= None
        self.screen 				= None
        self.cards 					= None
        self.buttons 				= None
        self.new_card_data 			= None
        self.new_chip_data 			= None
        self.yellow_box_x 			= None
        self.yellow_box_y 			= None
        self.yellow_box_width 		= None
        self.yellow_box_height 		= None
        self.name 					= None
        self.name_pos 				= None
        self.bet_pos 				= None
        self.player_bet 			= None
        self.player_fold 			= None
        self.current_player_num 	= None
        self.num_players 			= None
        self.dealing_card_num 		= None
        self.last_dealing_card_num 	= None
        self.new_card_added			= None
        self.game_in_progress 		= None
        self.play_again 			= None
        self.hand_in_progress		= None
        self.player_hand			= None
        self.player_purse           = None
        self.font_height			= None
        
        

