#!/usr/bin/env ruby
puml = []
picked = false
File.open('./README.md','r') do |f|
	f.each_line do |l|
	  l.chomp!
      if picked
        puml << l
        break if l == '@enduml'
        next
      end

      if l == '@startuml'
        picked = true
        puml << l
      end
    end
end
File.open('./tmp.puml','w') do |f|
  puml.each do |l|
    f.puts(l)
  end
end

r = `java -jar #{ENV['HOME']}/bin/plantuml.jar ./tmp.puml`
puts r
File.delete('./tmp.puml')
