import os
import sys
import pygame
import time

# Local imports
my_modules_path = os.getcwd()+"/includes"
if sys.path[0] != my_modules_path:
    sys.path.insert(0, my_modules_path)
    
from common import *
from globals import *
from button_handler import *
from actions import *


def draw_cards(screen,
               player_pos_start,
               hand,
               show_card):

    drawn = False
    if show_card > len(hand):
        return drawn
    
    common_vars   = CommonVariables.get_instance()

    player_x_pos, player_y_pos, horiz, vert = player_pos_start
    image_db = ImageDB.get_instance()
 
    for index_y, card in enumerate(hand):
        key = card
        if index_y == show_card-1:
            if key == "??":
                filename = common_vars.cards.face_down
            else:
                filename = common_vars.cards.cards[key]
        
            screen.blit(image_db.get_image(filename), (player_x_pos, player_y_pos))
            drawn = True
                
        player_x_pos += horiz
        player_y_pos -= vert

        x_offset = -50
        y_offset = -40  
    
    return drawn

def draw_button(screen, button_x_pos, button_y_pos, button, button_text, routine):

    common_vars   = CommonVariables.get_instance()
    image_db = ImageDB.get_instance()
    
    active_button    = IMAGE_PATH_BUTTONS + ACTIVE_BUTTON_FILENAME
    
    add_button(button_x_pos, button_y_pos,
               button_x_pos+common_vars.button_image_width,
               button_y_pos+common_vars.button_image_height,
               routine,
               button)
    filename = active_button
    
    screen.blit(image_db.get_image(filename), (button_x_pos, button_y_pos))
    
    text = common_vars.text_font.render('{0}'.format(button_text), False, BLACK_COLOR)
    size = text.get_size()       
    common_vars.screen.blit(text, (button_x_pos+(int(common_vars.button_image_width/2) - int(size[0]/2)), button_y_pos + int(common_vars.button_image_height/2)-int(size[1]/2)))

def draw_buttons(screen, buttons):

    common_vars   = CommonVariables.get_instance()
    button_x_pos, button_y_pos = BUTTONS_START_POS
    
    for button in buttons:

        draw_button(screen, button_x_pos, button_y_pos, button, buttons[button], send_action_to_server )
        button_x_pos += GAP_BETWEEN_BUTTONS + common_vars.button_image_width
        
        if button_x_pos + common_vars.button_image_width > common_vars.player_card_start_pos[0][0]:
            button_x_pos = BUTTONS_START_POS_X
            button_y_pos += common_vars.button_image_height
        

    
def draw_chips(screen, player_cash,
               chips_image_width,
               visible):
    
    common_vars   = CommonVariables.get_instance()
    sound_db = SoundDB.get_instance()
    chip_sound = sound_db.get_sound(SOUND_PATH + 'chipsstack.wav')
    
    chips_x_pos, chips_y_pos = CHIPS_START_POS
    gap = chips_image_width + GAP_BETWEEN_CHIPS
    image_db = ImageDB.get_instance()
    if visible:
        if player_cash >= 5:
            add_button(chips_x_pos, chips_y_pos, chips_x_pos+common_vars.chips_image_width,chips_y_pos+common_vars.chips_image_height,
                   bet_lower_button)
            screen.blit(image_db.get_image(IMAGE_PATH_CHIPS + CHIP_5_FILENAME_ON),
                        (chips_x_pos, chips_y_pos))
            
        if player_cash >= 10:
            chips_x_pos += gap
            add_button(chips_x_pos, chips_y_pos, chips_x_pos+common_vars.chips_image_width,chips_y_pos+common_vars.chips_image_height,
                   bet_higher_button)
            screen.blit(image_db.get_image(IMAGE_PATH_CHIPS + CHIP_10_FILENAME_ON),
                        (chips_x_pos, chips_y_pos))
    else:
        if player_cash >= 5:
            screen.blit(image_db.get_image(IMAGE_PATH_CHIPS + CHIP_5_FILENAME_OFF),
                        (chips_x_pos, chips_y_pos))
        if player_cash >= 10:
            chips_x_pos += gap
            screen.blit(image_db.get_image(IMAGE_PATH_CHIPS + CHIP_10_FILENAME_OFF),
                        (chips_x_pos, chips_y_pos))            


def draw_bet_in_progress(screen, player_bet, chips_image_width):
    
    image_db = ImageDB.get_instance()
    common_vars   = CommonVariables.get_instance()
    
    chip_x_pos = common_vars.yellow_box_x
    chip_y_pos = common_vars.yellow_box_y
    chip_y_pos += 4
    chip_x_pos += common_vars.yellow_box_width/2 - chips_image_width/2
    for chip in common_vars.player_bets:
        screen.blit(image_db.get_image(IMAGE_PATH_CHIPS + 'chip_{0}_w85h85.png'.format(chip)),
                    (chip_x_pos, chip_y_pos))
        chip_y_pos += 8


def draw_purse(screen, player_num):
    common_vars   = CommonVariables.get_instance()
    
    x_pos, y_pos = common_vars.purse_pos[player_num]
    
    message1 = common_vars.text_font.render('{0}'.format(common_vars.player_purse[player_num]), False, YELLOW_COLOR)
    common_vars.screen.blit(message1, (x_pos, y_pos))

def draw_pot(screen):
    
    common_vars   = CommonVariables.get_instance()
    total = common_vars.server.get_pot()
    
     
    
    
    message1 = common_vars.text_font.render('{0}'.format(total), False, BLACK_COLOR)
    y_pos = GAME_BOARD_Y_SIZE / 2  - common_vars.font_height / 2
    x_pos = GAME_BOARD_X_SIZE / 2  - message1.get_rect()[2] / 2
    common_vars.screen.blit(message1, (x_pos, y_pos))
    
def draw_name(screen, player_num):
    common_vars   = CommonVariables.get_instance()
    
    common_vars.text_font.set_bold(player_num == common_vars.current_player_num)
    common_vars.text_font.set_italic(player_num == common_vars.current_player_num)
    
    x_pos, y_pos = common_vars.name_pos[player_num]
    
    
    if player_num == common_vars.current_player_num:
        text = common_vars.text_font.render(' {0}  '.format(common_vars.name[player_num]), False, BLACK_COLOR)
        size = text.get_size()       
    else:
        text = common_vars.text_font.render('  {0}  '.format(common_vars.name[player_num]), False, YELLOW_COLOR)
        size = text.get_size()
        size = (size[0]+20, size[1])
    
    temp_surface = pygame.Surface(size)
    if player_num == common_vars.current_player_num:     
        temp_surface.fill(GREY_COLOR)
    else:
        temp_surface.fill(GAME_BOARD_COLOR)
        
    temp_surface.blit(text, (0, 0))
    common_vars.screen.blit(temp_surface, (x_pos, y_pos))
    
    common_vars.text_font.set_bold(False)
    common_vars.text_font.set_italic(False)
        
