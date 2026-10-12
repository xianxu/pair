// side by side: real | A | B (+ diff row)
const {PNG}=require('pngjs');const fs=require('fs');const [dir,i]=process.argv.slice(2);
const files=[dir+'/shots/'+i+'.png',dir+'/diff/A-'+i+'.png',dir+'/diff/B-'+i+'.png'];
const ims=files.map(f=>fs.existsSync(f)?PNG.sync.read(fs.readFileSync(f)):null);
const w=ims[0].width,h=ims[0].height;const o=new PNG({width:w*3+8,height:h});o.data.fill(255);
ims.forEach((im,k)=>{if(im)for(let y=0;y<h;y++)im.data.copy(o.data,((y*o.width)+k*(w+4))*4,y*w*4,(y+1)*w*4)});
fs.writeFileSync(dir+'/sbs-'+i+'.png',PNG.sync.write(o));
