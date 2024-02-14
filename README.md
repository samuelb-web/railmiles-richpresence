# RailMiles Discord Rich Presence
This application pulls your RailMiles data (individual or league) and displays it as a Rich Presence on your Discord profile.

## Installation
1. Download the .zip for your platform from the [Releases](https://github.com/samuelb-web/railmiles-richpresence/releases) page.
2. Unzip into a folder on your system
3. Update the `config.yaml` file with your RailMiles information
4. Run the executable


🎉 You should now have your RailMiles data on Discord!

## Notes
- Data will automatically be updated at xx:15 and xx:45 every hour
- League data will only be pulled hourly due to restrictions with RailMiles
- The RailMiles URL inside of your config.yaml file will need to contain https:// and a trailing / (https://username.railmiles.me/)