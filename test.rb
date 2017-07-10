#!/usr/bin/env ruby

require 'net/http'
require 'uri'
require 'json'

happy_test_cases = [
  # buy 42.45 LTC with USD
  {'action'         => 'buy',
   'base_currency'  => 'LTC',
   'quote_currency' => 'USD',
   'amount'         => 42.45},

  # sell 42.31 LTC for USD
  {'action'         => 'sell',
   'base_currency'  => 'LTC',
   'quote_currency' => 'USD',
   'amount'         => 42.45},

  # buy 20.35 BTC with USD
  {'action'         => 'buy',
   'base_currency'  => 'BTC',
   'quote_currency' => 'USD',
   'amount'         => 20.35},

  # buy 10 ETH with BTC
  {'action'         => 'buy',
   'base_currency'  => 'BTC',
   'quote_currency' => 'ETH',
   'amount'         => 10},

  # buy 10 BTC with ETH
  {'action'         => 'buy',
   'base_currency'  => 'ETH',
   'quote_currency' => 'BTC',
   'amount'         => 10},

  # buy 100 USD with BTC
  {'action'         => 'buy',
   'base_currency'  => 'USD',
   'quote_currency' => 'BTC',
   'amount'         => 100},
]

sad_test_cases = [
  # buy 100 LTC with GBP
  {'action'         => 'buy',
   'base_currency'  => 'LTC',
   'quote_currency' => 'GBP',
   'amount'         => 100},

  # buy 100 ARK with USD
  {'action'         => 'buy',
   'base_currency'  => 'ARK',
   'quote_currency' => 'USD',
   'amount'         => 100},

  # sell 100 USD for Ark
  {'action'         => 'sell',
   'base_currency'  => 'USD',
   'quote_currency' => 'Ark',
   'amount'         => 100},

  # waffle 100 BTC with USD
  {'action'         => 'waffle',
   'base_currency'  => 'BTC',
   'quote_currency' => 'USD',
   'amount'         => 100},

  # buy -100 BTC with USD
  {'action'         => 'buy',
   'base_currency'  => 'BTC',
   'quote_currency' => 'USD',
   'amount'         => -100},

  # sell 25,000,000 BTC for USD
  {'action'         => 'sell',
   'base_currency'  => 'BTC',
   'quote_currency' => 'USD',
   'amount'         => 25000000},
]

def decimals(a)
  a.to_s.split(".")[1].size
end

def acceptable(request, response, quote, amount)
  fails = 0

  if response.code == "200"
    print '.'
  else
    puts "FAIL: API responded with error status #{response.code}"
    fails += 1
  end

  if quote['currency'] == request['quote_currency']
    print '.'
  else
    fails += 1
    puts 'FAIL: returned currency should be requested quote currency'
    puts "Expected '#{request['quote_currency']}' got '#{quote['currency']}'"
  end

  price = quote['price'].to_f
  total = quote['total'].to_f

  precision = 2
  if ['BTC', 'ETH', 'LTC'].include? quote['currency']
    precision = 8
  end

  expected_total = (amount * price).round(precision)
  if expected_total == total
    print '.'
  else
    fails += 1
    puts 'FAIL: quote total should be the amount requested times the price'
    puts "Expected '#{total}' got '#{expected_total}'"
  end

  if decimals(total) <= precision
    print '.'
  else
    fails += 1
    puts "FAIL: total is too precise, it has #{decimals(total)} but should have fewer than 8"
  end

  return fails
end

def unacceptable(request, response, quote, amount)
  fails = 0

  if response.code != "200"
    print '.'
  else
    puts "FAIL: Bad API request responded with success status"
    fails += 1
  end

  message = quote['message']
  if message && message.length > 0
    print '.'
  else
    puts "FAIL: Bad API request did not respond with error message"
  end

  return fails
end

def run_test(http, test_case)
  amount = test_case['amount']
  request = test_case.merge('amount' => amount.to_s)
  puts "--> #{request}"

  response = http.post '/quote', request.to_json,
    'Content-Type' => 'application/json'

  quote = JSON.parse(response.body)
  puts "<-- #{quote}"
  return [request, response, quote, amount]
end

port = (ENV['GDAX_QUOTE_LISTEN_PORT'] || 3000).to_i
Net::HTTP.start('localhost', port) do |http|
  fails = 0

  happy_test_cases.each do |test_case|
    result = run_test(http, test_case)
    fails += acceptable(*result)
    puts "\n"
  end

  sad_test_cases.each do |test_case|
    result = run_test(http, test_case)
    fails += unacceptable(*result)
    puts "\n"
  end

  if fails == 0
    puts "Test suite passed, to the moon!!"
    exit 0
  else
    puts "#{fails} assertions failed"
    exit 1
  end
end
