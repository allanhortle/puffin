-- globals
previousApp = hs.application.frontmostApplication();

-- Reload config
hs.hotkey.bind({"cmd", "alt", "ctrl"}, "R", function()
    hs.reload()
    hs.alert.show("Config loaded")
end)


-- Show and hide the finder
function showFinder()
    local finder = hs.appfinder.appFromName("Finder");
    local currentApp = hs.application.frontmostApplication();
    local bringWindowsToFront = {"Window", "Bring All to Front"};

    if currentApp == finder then
        hs.application.launchOrFocus(previousApp:name());
    else
        previousApp = currentApp;
        hs.application.launchOrFocus("Finder");
        finder:selectMenuItem(bringWindowsToFront);
    end
end
hs.hotkey.bind({"alt"}, 'tab', showFinder);


function log(str)
    local screenW = hs.screen.mainScreen():frame().w;
    local geom    = require "hs.geometry"
    local rectify = geom.rect
    local my_ls_spot = rectify(100,100,1000,1000)

    if text then
        text:delete()
        rectangle:delete()
        if textTimer then
            textTimer:stop()
        end
    end

    rectangle = hs.drawing.rectangle(rectify(0,0,screenW,120))
    rectangle:setFill(true)
    rectangle:setFillColor({["red"]=0,["blue"]=0,["green"]=0,["alpha"]=.8})
    rectangle:show()

    text = hs.drawing.text(my_ls_spot, str)
    text:show()

    -- Set a timer to delete the circle after 3 seconds
    textTimer = hs.timer.doAfter(2, function() 
        text:delete() 
        rectangle:delete() 
    end)
end
hs.hotkey.bind({"cmd", "alt", "ctrl"}, "D", function() return log('rad') end)