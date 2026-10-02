import os.path

application = defines["app"]
appname = os.path.basename(application)
here = os.path.dirname(os.path.abspath(defines["settings"]))

format = "ULMO"
filesystem = "HFS+"
files = [application]
symlinks = {"Applications": "/Applications"}
hide_extensions = [appname]
icon = os.path.join(application, "Contents", "Resources", "iconfile.icns")

background = os.path.join(here, "dmg-background.png")
window_rect = ((200, 140), (660, 428))
default_view = "icon-view"
show_status_bar = False
show_tab_view = False
show_toolbar = False
show_pathbar = False
show_sidebar = False
show_icon_preview = False
include_icon_view_settings = True
arrange_by = None
icon_size = 128
text_size = 13
icon_locations = {appname: (165, 170), "Applications": (495, 170)}
