import os
import shutil
import hashlib
import yaml
import markdown
import rcssmin
from datetime import datetime
from jinja2 import Environment, FileSystemLoader


def build_item_lookup(data):
    lookup = {}
    sources = [
        ('primaryPublications', 'publication', 'primary'),
        ('secondaryPublications', 'publication', 'secondary'),
        ('primaryProjects', 'project', 'primary'),
        ('secondaryProjects', 'project', 'secondary'),
        ('courses', 'teaching', 'primary'),
        ('supervision', 'teaching', 'primary'),
        ('presentations', 'presentation', 'secondary'),
    ]
    for key, item_type, subtype in sources:
        for item in data.get(key, []):
            if 'id' in item:
                lookup[item['id']] = {**item, '_type': item_type, '_subtype': subtype}
    return lookup


def resolve_subsections(subsections, lookup):
    for section in subsections:
        section['resolved_items'] = []
        for item_id in section.get('items', []):
            if item_id in lookup:
                section['resolved_items'].append(lookup[item_id])
            else:
                print(f"Warning: item '{item_id}' not found in lookup")


def fingerprint(content_bytes):
    return hashlib.sha256(content_bytes).hexdigest()[:8]


def hashed_name(filename, content_bytes):
    name, ext = os.path.splitext(filename)
    return f'{name}.{fingerprint(content_bytes)}{ext}'


def copy_assets(src_dir='asset', dst_dir='dist'):
    asset_map = {}
    deferred = []

    for root, dirs, files in os.walk(src_dir):
        rel_dir = os.path.relpath(root, src_dir)
        target_dir = os.path.join(dst_dir, rel_dir) if rel_dir != '.' else dst_dir
        os.makedirs(target_dir, exist_ok=True)
        for filename in files:
            src_path = os.path.join(root, filename)
            rel_path = os.path.join(rel_dir, filename) if rel_dir != '.' else filename

            if filename.endswith('.webmanifest'):
                deferred.append((src_path, target_dir, rel_path, filename))
                continue

            if filename.endswith('.css'):
                with open(src_path, 'r') as f:
                    content = rcssmin.cssmin(f.read()).encode()
            else:
                with open(src_path, 'rb') as f:
                    content = f.read()

            out_name = hashed_name(filename, content)
            dst_path = os.path.join(target_dir, out_name)
            with open(dst_path, 'wb') as f:
                f.write(content)
            asset_map[rel_path] = os.path.join(rel_dir, out_name) if rel_dir != '.' else out_name
            print(f'Hashed {src_path} -> {dst_path}')

    for src_path, target_dir, rel_path, filename in deferred:
        with open(src_path, 'r') as f:
            content = f.read()
        for original, hashed in asset_map.items():
            content = content.replace(f'/{original}', f'/{hashed}')
        content_bytes = content.encode()
        out_name = hashed_name(filename, content_bytes)
        dst_path = os.path.join(target_dir, out_name)
        with open(dst_path, 'w') as f:
            f.write(content)
        asset_map[rel_path] = out_name
        print(f'Hashed {src_path} -> {dst_path}')

    return asset_map


if __name__ == '__main__':
    with open('data.yaml', 'r') as file:
        profile_data = yaml.safe_load(file)

    # Build item lookup and resolve index page subsection references
    item_lookup = build_item_lookup(profile_data)
    resolve_subsections(profile_data.get('researchSections', []), item_lookup)
    resolve_subsections(profile_data.get('teachingSections', []), item_lookup)

    # Render markdown content files
    md = markdown.Markdown()
    content = {}
    content_dir = 'content'
    if os.path.isdir(content_dir):
        for filename in os.listdir(content_dir):
            if filename.endswith('.md'):
                with open(os.path.join(content_dir, filename), 'r') as f:
                    name = filename[:-3]
                    content[name] = md.convert(f.read())
                    md.reset()

    env = Environment(loader=FileSystemLoader('templates'))
    current_year = datetime.now().year

    os.makedirs('dist', exist_ok=True)
    os.makedirs('dist/publications', exist_ok=True)
    os.makedirs('dist/projects', exist_ok=True)
    os.makedirs('dist/activities', exist_ok=True)

    assets = copy_assets()

    def render_template(template_name, output_path, **kwargs):
        template = env.get_template(template_name)
        html = template.render(year=current_year, assets=assets, **kwargs)

        with open(output_path, 'w') as file:
            file.write(html)

        print(f'Generated {output_path}')

    render_template('index.html', 'dist/index.html', data=profile_data, content=content, is_home_page=True)
    render_template('publications.html', 'dist/publications/index.html', data=profile_data, is_home_page=False)
    render_template('projects.html', 'dist/projects/index.html', data=profile_data, is_home_page=False)
    render_template('activities.html', 'dist/activities/index.html', data=profile_data, is_home_page=False)

    # Generate sitemap.xml and robots.txt
    base_url = profile_data['baseUrl']
    today = datetime.now().strftime('%Y-%m-%d')
    pages = [{'path': p, 'lastmod': today} for p in ['/', '/publications/', '/projects/', '/activities/']]
    for tmpl, output in [('sitemap.xml', 'dist/sitemap.xml'), ('robots.txt', 'dist/robots.txt')]:
        with open(output, 'w') as f:
            f.write(env.get_template(tmpl).render(base_url=base_url, pages=pages))
        print(f'Generated {output}')

    print('Static site generation complete!')
